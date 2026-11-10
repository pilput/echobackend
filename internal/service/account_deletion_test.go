package service

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/model"
)

// memoryDeletionTokens is an in-memory AccountDeletionTokenRepository.
type memoryDeletionTokens struct {
	tokens []*model.AccountDeletionToken
}

func (m *memoryDeletionTokens) Create(ctx context.Context, token *model.AccountDeletionToken) error {
	token.ID = "t-" + token.Token[:8]
	m.tokens = append(m.tokens, token)
	return nil
}

func (m *memoryDeletionTokens) FindByToken(ctx context.Context, token string) (*model.AccountDeletionToken, error) {
	for _, t := range m.tokens {
		if t.Token == token {
			copied := *t
			return &copied, nil
		}
	}
	return nil, nil
}

func (m *memoryDeletionTokens) MarkUsed(ctx context.Context, id string) (bool, error) {
	for _, t := range m.tokens {
		if t.ID == id && t.UsedAt == nil {
			now := time.Now()
			t.UsedAt = &now
			return true, nil
		}
	}
	return false, nil
}

func (m *memoryDeletionTokens) DeleteByUserID(ctx context.Context, userID string) error {
	kept := m.tokens[:0]
	for _, t := range m.tokens {
		if t.UserID != userID {
			kept = append(kept, t)
		}
	}
	m.tokens = kept
	return nil
}

type fakeEmailSender struct {
	configured bool
	to         string
	link       string
	err        error
}

func (f *fakeEmailSender) EnqueuePasswordResetEmail(to, resetLink string) error { return nil }
func (f *fakeEmailSender) EnqueueAccountDeletionEmail(to, confirmLink string) error {
	f.to, f.link = to, confirmLink
	return f.err
}
func (f *fakeEmailSender) IsConfigured() bool { return f.configured }

type deletionFixture struct {
	svc        AuthService
	tokens     *memoryDeletionTokens
	mail       *fakeEmailSender
	deleted    []string
	revokedFor []string
	activities []string
}

func newDeletionFixture(t *testing.T) *deletionFixture {
	t.Helper()
	f := &deletionFixture{
		tokens: &memoryDeletionTokens{},
		mail:   &fakeEmailSender{configured: true},
	}
	userRepo := &mockUserRepo{
		getByIDFn: func(ctx context.Context, id string, deletedOnly bool) (*model.User, error) {
			if id != "u-1" {
				return nil, apperrors.ErrUserNotFound
			}
			return &model.User{ID: "u-1", Email: "user@example.com"}, nil
		},
		softDeleteFn: func(ctx context.Context, id string) error {
			f.deleted = append(f.deleted, id)
			return nil
		},
	}
	sessionRepo := &mockSessionRepo{
		deleteByUserIDFn: func(ctx context.Context, userID string) (int64, error) {
			f.revokedFor = append(f.revokedFor, userID)
			return 1, nil
		},
	}
	activity := &mockActivityService{
		logActivityFn: func(ctx context.Context, userID *string, activityType, status, ipAddress, userAgent string, errorMessage *string, metadata map[string]any) {
			f.activities = append(f.activities, activityType+":"+status)
		},
	}
	cfg := testAuthConfig()
	cfg.Frontend.DeleteAccountURL = "https://pilput.example/confirm-delete-account"
	f.svc = NewAuthService(&mockAuthRepo{}, userRepo, sessionRepo, nil, f.tokens, activity, cfg, nil, f.mail)
	return f
}

// emailedToken returns the raw token from the link in the last email.
func (f *deletionFixture) emailedToken(t *testing.T) string {
	t.Helper()
	parsed, err := url.Parse(f.mail.link)
	if err != nil {
		t.Fatalf("emailed link is not a URL: %v", err)
	}
	return parsed.Query().Get("token")
}

func TestRequestAccountDeletion_EmailsLinkAndDeletesNothing(t *testing.T) {
	f := newDeletionFixture(t)

	if err := f.svc.RequestAccountDeletion(context.Background(), "u-1", "1.2.3.4", "ua"); err != nil {
		t.Fatalf("RequestAccountDeletion failed: %v", err)
	}

	if f.mail.to != "user@example.com" {
		t.Fatalf("email sent to %q, want user@example.com", f.mail.to)
	}
	if !strings.HasPrefix(f.mail.link, "https://pilput.example/confirm-delete-account?token=ad_") {
		t.Fatalf("unexpected link %q", f.mail.link)
	}
	if len(f.deleted) != 0 || len(f.revokedFor) != 0 {
		t.Fatal("requesting deletion must not delete the account or revoke sessions")
	}
	if len(f.tokens.tokens) != 1 {
		t.Fatalf("expected 1 stored token, got %d", len(f.tokens.tokens))
	}
	stored := f.tokens.tokens[0]
	if stored.Token == f.emailedToken(t) {
		t.Fatal("token must be stored hashed, not raw")
	}
	if ttl := time.Until(stored.ExpiresAt); ttl <= 25*time.Minute || ttl > 30*time.Minute {
		t.Fatalf("unexpected token lifetime %v", ttl)
	}
}

func TestRequestAccountDeletion_NewRequestReplacesOldLink(t *testing.T) {
	f := newDeletionFixture(t)
	ctx := context.Background()

	_ = f.svc.RequestAccountDeletion(ctx, "u-1", "", "")
	oldToken := f.emailedToken(t)
	_ = f.svc.RequestAccountDeletion(ctx, "u-1", "", "")

	if len(f.tokens.tokens) != 1 {
		t.Fatalf("expected only the newest token to remain, got %d", len(f.tokens.tokens))
	}
	if err := f.svc.ConfirmAccountDeletion(ctx, oldToken, "", ""); !errors.Is(err, apperrors.ErrInvalidToken) {
		t.Fatalf("old link should be invalid, got %v", err)
	}
}

func TestRequestAccountDeletion_EmailNotConfigured(t *testing.T) {
	f := newDeletionFixture(t)
	f.mail.configured = false

	err := f.svc.RequestAccountDeletion(context.Background(), "u-1", "", "")
	if !errors.Is(err, apperrors.ErrEmailNotConfigured) {
		t.Fatalf("expected ErrEmailNotConfigured, got %v", err)
	}
	if len(f.tokens.tokens) != 0 {
		t.Fatal("no token should be created when email cannot be sent")
	}
}

func TestRequestAccountDeletion_QueueFailureIsReported(t *testing.T) {
	f := newDeletionFixture(t)
	f.mail.err = errors.New("queue full")

	if err := f.svc.RequestAccountDeletion(context.Background(), "u-1", "", ""); err == nil {
		t.Fatal("expected an error when the email cannot be queued")
	}
}

func TestConfirmAccountDeletion_DeletesAndRevokesSessions(t *testing.T) {
	f := newDeletionFixture(t)
	ctx := context.Background()
	_ = f.svc.RequestAccountDeletion(ctx, "u-1", "", "")

	if err := f.svc.ConfirmAccountDeletion(ctx, f.emailedToken(t), "", ""); err != nil {
		t.Fatalf("ConfirmAccountDeletion failed: %v", err)
	}

	if len(f.deleted) != 1 || f.deleted[0] != "u-1" {
		t.Fatalf("expected u-1 to be soft-deleted, got %v", f.deleted)
	}
	if len(f.revokedFor) != 1 || f.revokedFor[0] != "u-1" {
		t.Fatalf("expected sessions of u-1 to be revoked, got %v", f.revokedFor)
	}
	last := f.activities[len(f.activities)-1]
	if last != model.ActivityAccountDeleted+":"+model.StatusSuccess {
		t.Fatalf("expected account_deleted activity, got %q", last)
	}
}

func TestConfirmAccountDeletion_TokenWorksOnce(t *testing.T) {
	f := newDeletionFixture(t)
	ctx := context.Background()
	_ = f.svc.RequestAccountDeletion(ctx, "u-1", "", "")
	token := f.emailedToken(t)

	if err := f.svc.ConfirmAccountDeletion(ctx, token, "", ""); err != nil {
		t.Fatalf("first confirm failed: %v", err)
	}
	if err := f.svc.ConfirmAccountDeletion(ctx, token, "", ""); !errors.Is(err, apperrors.ErrAccountDeletionTokenUsed) {
		t.Fatalf("expected ErrAccountDeletionTokenUsed, got %v", err)
	}
	if len(f.deleted) != 1 {
		t.Fatalf("account should be deleted exactly once, got %d", len(f.deleted))
	}
}

func TestConfirmAccountDeletion_ExpiredToken(t *testing.T) {
	f := newDeletionFixture(t)
	ctx := context.Background()
	_ = f.svc.RequestAccountDeletion(ctx, "u-1", "", "")
	f.tokens.tokens[0].ExpiresAt = time.Now().Add(-time.Minute)

	err := f.svc.ConfirmAccountDeletion(ctx, f.emailedToken(t), "", "")
	if !errors.Is(err, apperrors.ErrAccountDeletionTokenExpired) {
		t.Fatalf("expected ErrAccountDeletionTokenExpired, got %v", err)
	}
	if len(f.deleted) != 0 {
		t.Fatal("expired token must not delete the account")
	}
}

func TestConfirmAccountDeletion_UnknownToken(t *testing.T) {
	f := newDeletionFixture(t)

	err := f.svc.ConfirmAccountDeletion(context.Background(), "ad_unknown", "", "")
	if !errors.Is(err, apperrors.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}
