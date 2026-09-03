package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"html"
	"mime/multipart"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strings"
	"sync"
	"time"

	"echobackend/config"
	"echobackend/pkg/applog"
)

var log = applog.Component("email")

type passwordResetPayload struct {
	To        string `json:"to"`
	ResetLink string `json:"reset_link"`
}

// Service sends application emails through SMTP using an in-process worker pool.
type Service struct {
	host     string
	port     int
	username string
	password string
	from     string
	timeout  time.Duration
	taskTTL  time.Duration
	useTLS   bool

	tasks     chan passwordResetPayload
	quit      chan struct{}
	wg        sync.WaitGroup
	closeOnce sync.Once
}

// NewService creates an SMTP-backed email service and starts its background worker pool.
func NewService(cfg config.EmailConfig) *Service {
	service := &Service{
		host:     cfg.SMTPHost,
		port:     cfg.SMTPPort,
		username: cfg.SMTPUsername,
		password: cfg.SMTPPassword,
		from:     cfg.From,
		timeout:  cfg.Timeout,
		taskTTL:  cfg.TaskTimeout,
		useTLS:   cfg.UseTLS,
		tasks:    make(chan passwordResetPayload, 128),
		quit:     make(chan struct{}),
	}
	if service.hasSMTPConfig() {
		service.startWorkers(2)
		log.Info("email background workers started", "workers", 2)
	}
	return service
}

func (s *Service) startWorkers(workerCount int) {
	for i := 0; i < workerCount; i++ {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			for {
				select {
				case <-s.quit:
					return
				case payload, ok := <-s.tasks:
					if !ok {
						return
					}
					taskCtx := context.Background()
					var cancel context.CancelFunc
					if s.taskTTL > 0 {
						taskCtx, cancel = context.WithTimeout(taskCtx, s.taskTTL)
					}
					if err := s.SendPasswordResetEmail(taskCtx, payload.To, payload.ResetLink); err != nil {
						log.Error("failed to send password reset email", "error", err, "to", payload.To)
					}
					if cancel != nil {
						cancel()
					}
				}
			}
		}()
	}
}

// IsConfigured reports whether email delivery is enabled.
func (s *Service) IsConfigured() bool {
	return s != nil && s.hasSMTPConfig()
}

func (s *Service) hasSMTPConfig() bool {
	return s != nil && s.host != "" && s.port > 0 && s.from != ""
}

// Close stops background workers gracefully.
func (s *Service) Close() error {
	if s == nil {
		return nil
	}
	s.closeOnce.Do(func() {
		close(s.quit)
		s.wg.Wait()
	})
	return nil
}

// EnqueuePasswordResetEmail queues a password reset email for background delivery.
func (s *Service) EnqueuePasswordResetEmail(to, resetLink string) error {
	if !s.IsConfigured() {
		return errors.New("email service not configured")
	}

	payload := passwordResetPayload{To: to, ResetLink: resetLink}
	select {
	case s.tasks <- payload:
		return nil
	default:
		return errors.New("email queue is full")
	}
}

// SendPasswordResetEmail sends the password reset link email.
func (s *Service) SendPasswordResetEmail(ctx context.Context, to, resetLink string) error {
	if !s.hasSMTPConfig() {
		return errors.New("email service not configured")
	}

	text, htmlBody := passwordResetTemplate(resetLink, "1 hour")
	return s.send(ctx, to, "Reset your password", text, htmlBody)
}

func (s *Service) send(ctx context.Context, to, subject, textBody, htmlBody string) error {
	message, err := buildMessage(s.from, to, subject, textBody, htmlBody)
	if err != nil {
		return err
	}

	address := fmt.Sprintf("%s:%d", s.host, s.port)
	dialer := net.Dialer{Timeout: s.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("smtp dial %s failed: %w", address, err)
	}
	defer func() { _ = conn.Close() }()
	if s.timeout > 0 {
		if err := conn.SetDeadline(time.Now().Add(s.timeout)); err != nil {
			return fmt.Errorf("smtp set deadline failed: %w", err)
		}
	}

	var client *smtp.Client
	if s.useTLS {
		tlsConn := tls.Client(conn, &tls.Config{ServerName: s.host, MinVersion: tls.VersionTLS12})
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			return fmt.Errorf("smtp tls handshake failed: %w", err)
		}
		client, err = smtp.NewClient(tlsConn, s.host)
	} else {
		client, err = smtp.NewClient(conn, s.host)
	}
	if err != nil {
		return fmt.Errorf("smtp client init failed: %w", err)
	}
	defer func() { _ = client.Close() }()

	if !s.useTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			tlsConfig := &tls.Config{ServerName: s.host, MinVersion: tls.VersionTLS12}
			if err := client.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("smtp starttls failed: %w", err)
			}
		}
	}

	if s.username != "" || s.password != "" {
		auth := smtp.PlainAuth("", s.username, s.password, s.host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth failed: %w", err)
		}
	}

	fromAddress, err := parseAddress(s.from)
	if err != nil {
		return fmt.Errorf("invalid smtp from address: %w", err)
	}
	toAddress, err := parseAddress(to)
	if err != nil {
		return fmt.Errorf("invalid smtp recipient address: %w", err)
	}

	if err := client.Mail(fromAddress); err != nil {
		return fmt.Errorf("smtp mail from failed: %w", err)
	}
	if err := client.Rcpt(toAddress); err != nil {
		return fmt.Errorf("smtp rcpt to failed: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data command failed: %w", err)
	}
	if _, err := writer.Write(message); err != nil {
		_ = writer.Close()
		return fmt.Errorf("smtp write message failed: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("smtp close message failed: %w", err)
	}

	if err := client.Quit(); err != nil {
		return fmt.Errorf("smtp quit failed: %w", err)
	}

	return nil
}

func buildMessage(from, to, subject, textBody, htmlBody string) ([]byte, error) {
	fromAddress, err := mail.ParseAddress(from)
	if err != nil {
		return nil, err
	}
	toAddress, err := mail.ParseAddress(to)
	if err != nil {
		return nil, err
	}

	var body bytes.Buffer
	multipartWriter := multipart.NewWriter(&body)

	textPart, err := multipartWriter.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {"text/plain; charset=utf-8"},
		"Content-Transfer-Encoding": {"8bit"},
	})
	if err != nil {
		return nil, err
	}
	if _, err := textPart.Write([]byte(textBody)); err != nil {
		return nil, err
	}

	htmlPart, err := multipartWriter.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {"text/html; charset=utf-8"},
		"Content-Transfer-Encoding": {"8bit"},
	})
	if err != nil {
		return nil, err
	}
	if _, err := htmlPart.Write([]byte(htmlBody)); err != nil {
		return nil, err
	}

	if err := multipartWriter.Close(); err != nil {
		return nil, err
	}

	var message bytes.Buffer
	headers := [][2]string{
		{"From", fromAddress.String()},
		{"To", toAddress.String()},
		{"Subject", sanitizeHeader(subject)},
		{"MIME-Version", "1.0"},
		{"Content-Type", fmt.Sprintf("multipart/alternative; boundary=%q", multipartWriter.Boundary())},
	}
	for _, header := range headers {
		message.WriteString(header[0])
		message.WriteString(": ")
		message.WriteString(header[1])
		message.WriteString("\r\n")
	}
	message.WriteString("\r\n")
	message.Write(body.Bytes())

	return message.Bytes(), nil
}

func parseAddress(raw string) (string, error) {
	address, err := mail.ParseAddress(raw)
	if err != nil {
		return "", err
	}
	return address.Address, nil
}

func sanitizeHeader(value string) string {
	value = strings.ReplaceAll(value, "\r", "")
	return strings.ReplaceAll(value, "\n", "")
}

func passwordResetTemplate(resetLink, expiresIn string) (string, string) {
	escapedLink := html.EscapeString(resetLink)
	escapedExpiresIn := html.EscapeString(expiresIn)

	textBody := fmt.Sprintf(
		"We received a request to reset your password.\n\nReset your password here:\n%s\n\nThis link expires in %s. If you did not request a password reset, you can safely ignore this email.",
		resetLink,
		expiresIn,
	)

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Reset Your Password</title>
  <style>
    body { margin: 0; padding: 0; background: #ffffff; color: #111111; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Arial, sans-serif; }
    .page { width: 100%%; padding: 32px 16px; background: #ffffff; }
    .container { max-width: 560px; margin: 0 auto; background: #ffffff; border: 1px solid #e5e5e5; }
    .header { padding: 28px 32px 18px; border-bottom: 1px solid #eeeeee; }
    .brand { margin: 0 0 12px; color: #555555; font-size: 13px; font-weight: 600; }
    h1 { margin: 0; color: #111111; font-size: 23px; line-height: 1.3; font-weight: 600; }
    .content { padding: 26px 32px 32px; }\n    p { margin: 0 0 16px; color: #333333; font-size: 16px; line-height: 1.6; }
    .button-wrap { margin: 24px 0; }
    .button { display: inline-block; background: #111111; color: #ffffff; padding: 12px 20px; text-decoration: none; border-radius: 4px; font-size: 15px; font-weight: 600; }
    .meta { margin: 22px 0; padding: 14px 0; border-top: 1px solid #eeeeee; border-bottom: 1px solid #eeeeee; color: #555555; font-size: 14px; line-height: 1.5; }
    .fallback { margin-top: 18px; padding-top: 18px; border-top: 1px solid #eeeeee; }
    .fallback p { color: #666666; font-size: 13px; line-height: 1.5; }
    .link { color: #111111; word-break: break-all; overflow-wrap: anywhere; }
    .footer { max-width: 560px; margin: 16px auto 0; text-align: center; }
    .footer p { color: #777777; font-size: 12px; line-height: 1.5; }
    @media (max-width: 480px) {
      .page { padding: 16px 10px; }
      .header, .content { padding-left: 20px; padding-right: 20px; }
      h1 { font-size: 22px; }
      .button { display: block; text-align: center; }
    }
  </style>
</head>
<body>
  <div class="page">
    <div class="container">
      <div class="header">
        <p class="brand">Pilput</p>
        <h1>Reset your password</h1>
      </div>
      <div class="content">
        <p>We received a request to reset the password for your account. Use the button below to choose a new password.</p>
        <div class="button-wrap">
          <a href="%s" class="button">Reset password</a>
        </div>
        <div class="meta">This link expires in <strong>%s</strong>. For your security, do not forward this email or share the reset link.</div>
        <div class="fallback">
          <p>If the button does not work, copy and paste this link into your browser:</p>
          <p><a href="%s" class="link">%s</a></p>
        </div>
      </div>
    </div>
    <div class="footer">
      <p>If you did not request a password reset, you can safely ignore this email.</p>
    </div>
  </div>
</body>
</html>`, escapedLink, escapedExpiresIn, escapedLink, escapedLink)

	return textBody, htmlBody
}
