package openrouter

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"echobackend/config"
	"echobackend/pkg/applog"
)

var log = applog.Component("openrouter")

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type Choice struct {
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
}

type Response struct {
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Delta is one piece of a streamed answer. Exactly one field is set:
// Reasoning carries "thinking" text from reasoning models (never persisted),
// Content the answer itself.
type Delta struct {
	Content   string
	Reasoning string
}

type Client interface {
	GenerateResponse(ctx context.Context, messages []Message, model *string, temperature float64) (*Response, error)
	GenerateStream(ctx context.Context, messages []Message, model *string, temperature float64) (<-chan Delta, <-chan Usage, <-chan error)
}

type client struct {
	cfg     config.OpenRouterConfig
	timeout time.Duration
	// httpClient bounds whole non-streaming calls. streamClient has no overall
	// timeout: a streamed answer (especially with reasoning) can legitimately
	// run longer, so GenerateStream bounds the wait for the response head and
	// the gap between lines instead.
	httpClient   *http.Client
	streamClient *http.Client
}

func NewClient(cfg config.OpenRouterConfig) Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 90 * time.Second
	}
	return &client{
		cfg:          cfg,
		timeout:      timeout,
		httpClient:   &http.Client{Timeout: timeout},
		streamClient: &http.Client{},
	}
}

func (c *client) GenerateResponse(ctx context.Context, messages []Message, model *string, temperature float64) (*Response, error) {
	resp, err := c.callAPI(ctx, messages, model, false, temperature)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var payload Response
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode OpenRouter response: %w", err)
	}
	return &payload, nil
}

// errStreamTimedOut is reported when OpenRouter sends nothing (no response
// head, no line) for the client timeout.
var errStreamTimedOut = errors.New("OpenRouter stream timed out")

func (c *client) GenerateStream(ctx context.Context, messages []Message, model *string, temperature float64) (<-chan Delta, <-chan Usage, <-chan error) {
	deltas := make(chan Delta)
	usageCh := make(chan Usage, 1)
	errCh := make(chan error, 1)

	go func() {
		defer close(deltas)
		defer close(usageCh)
		defer close(errCh)

		// The idle timer cancels the request when OpenRouter goes quiet for
		// c.timeout; every line read pushes it back.
		streamCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		var timedOut atomic.Bool
		idle := time.AfterFunc(c.timeout, func() {
			timedOut.Store(true)
			cancel()
		})
		defer idle.Stop()
		fail := func(err error) {
			if timedOut.Load() {
				err = errStreamTimedOut
			}
			errCh <- err
		}

		resp, err := c.callAPI(streamCtx, messages, model, true, temperature)
		if err != nil {
			fail(err)
			return
		}
		defer func() { _ = resp.Body.Close() }()

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		var usage Usage
		for scanner.Scan() {
			idle.Reset(c.timeout)

			// SSE comment lines such as ": OPENROUTER PROCESSING" are ignored.
			data, ok := strings.CutPrefix(strings.TrimSpace(scanner.Text()), "data:")
			if !ok {
				continue
			}
			data = strings.TrimLeft(data, " ")
			if data == "[DONE]" {
				usageCh <- usage
				return
			}

			chunk, err := parseStreamData(data)
			if err != nil {
				errCh <- err
				return
			}
			if chunk.usage != nil {
				usage = *chunk.usage
			}
			for _, d := range chunk.deltas {
				select {
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				case deltas <- d:
				}
			}
		}

		if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
			if ctxErr := ctx.Err(); ctxErr != nil {
				errCh <- ctxErr
				return
			}
			fail(fmt.Errorf("failed to read OpenRouter stream: %w", err))
			return
		}
		usageCh <- usage
	}()

	return deltas, usageCh, errCh
}

type streamChunk struct {
	deltas []Delta
	usage  *Usage
}

// parseStreamData parses one SSE data payload of a chat-completions stream.
// Invalid JSON is skipped. Per the OpenRouter docs:
//   - reasoning streams in choices[0].delta.reasoning_details (items of type
//     reasoning.text / reasoning.summary; reasoning.encrypted has no readable
//     text). Some providers also send a plain delta.reasoning string; it is used
//     only when reasoning_details has no readable text, so the same reasoning
//     is never emitted twice.
//   - mid-stream failures arrive as a top-level error object (HTTP stays 200)
//     and are returned as the error.
func parseStreamData(data string) (streamChunk, error) {
	var payload struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
		Usage   *Usage `json:"usage"`
		Choices []struct {
			Delta struct {
				Content          *string `json:"content"`
				Reasoning        *string `json:"reasoning"`
				ReasoningDetails []struct {
					Type    string `json:"type"`
					Text    string `json:"text"`
					Summary string `json:"summary"`
				} `json:"reasoning_details"`
			} `json:"delta"`
		} `json:"choices"`
	}
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		return streamChunk{}, nil //nolint:nilerr // malformed chunks are skipped, not fatal
	}
	if payload.Error != nil {
		msg := payload.Error.Message
		if msg == "" {
			msg = "OpenRouter stream failed"
		}
		return streamChunk{}, errors.New(msg)
	}

	out := streamChunk{usage: payload.Usage}
	if len(payload.Choices) == 0 {
		return out, nil
	}
	delta := payload.Choices[0].Delta

	var reasoning strings.Builder
	for _, item := range delta.ReasoningDetails {
		switch item.Type {
		case "reasoning.text":
			reasoning.WriteString(item.Text)
		case "reasoning.summary":
			reasoning.WriteString(item.Summary)
		}
	}
	if reasoning.Len() == 0 && delta.Reasoning != nil {
		reasoning.WriteString(*delta.Reasoning)
	}
	if reasoning.Len() > 0 {
		out.deltas = append(out.deltas, Delta{Reasoning: reasoning.String()})
	}
	if delta.Content != nil && *delta.Content != "" {
		out.deltas = append(out.deltas, Delta{Content: *delta.Content})
	}
	return out, nil
}

func (c *client) callAPI(ctx context.Context, messages []Message, model *string, stream bool, temperature float64) (*http.Response, error) {
	apiKey := strings.TrimSpace(c.cfg.APIKey)
	if apiKey == "" {
		return nil, errors.New("OPENROUTER_API_KEY is not configured")
	}

	finalModel := strings.TrimSpace(c.cfg.DefaultModel)
	if model != nil && strings.TrimSpace(*model) != "" {
		finalModel = strings.TrimSpace(*model)
	}
	if finalModel == "" {
		return nil, errors.New("OPENROUTER_DEFAULT_MODEL is not configured")
	}
	if temperature < 0 || temperature > 2 {
		temperature = 0.7
	}

	body, err := json.Marshal(map[string]any{
		"model":       finalModel,
		"messages":    messages,
		"stream":      stream,
		"temperature": temperature,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encode OpenRouter request: %w", err)
	}

	url := strings.TrimRight(c.cfg.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenRouter request: %w", err)
	}
	log.Debug("sending chat completion request", "model", finalModel, "stream", stream)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.HTTPReferer != "" {
		req.Header.Set("HTTP-Referer", c.cfg.HTTPReferer)
	}
	if c.cfg.Title != "" {
		req.Header.Set("X-Title", c.cfg.Title)
	}

	httpClient := c.httpClient
	if stream {
		httpClient = c.streamClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OpenRouter request failed: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer func() { _ = resp.Body.Close() }()
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("OpenRouter API error: %s %s", resp.Status, strings.TrimSpace(string(payload)))
	}

	return resp, nil
}
