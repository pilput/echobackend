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

type Client interface {
	GenerateResponse(ctx context.Context, messages []Message, model *string, temperature float64) (*Response, error)
	GenerateStream(ctx context.Context, messages []Message, model *string, temperature float64) (<-chan string, <-chan Usage, <-chan error)
}

type client struct {
	cfg        config.OpenRouterConfig
	httpClient *http.Client
}

func NewClient(cfg config.OpenRouterConfig) Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 90 * time.Second
	}
	return &client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: timeout,
		},
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

func (c *client) GenerateStream(ctx context.Context, messages []Message, model *string, temperature float64) (<-chan string, <-chan Usage, <-chan error) {
	chunks := make(chan string)
	usageCh := make(chan Usage, 1)
	errCh := make(chan error, 1)

	go func() {
		defer close(chunks)
		defer close(usageCh)
		defer close(errCh)

		resp, err := c.callAPI(ctx, messages, model, true, temperature)
		if err != nil {
			errCh <- err
			return
		}
		defer func() { _ = resp.Body.Close() }()

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		var usage Usage
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				usageCh <- usage
				return
			}

			var payload struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
				Usage *Usage `json:"usage"`
			}
			if err := json.Unmarshal([]byte(data), &payload); err != nil {
				continue
			}
			if payload.Usage != nil {
				usage = *payload.Usage
			}
			if len(payload.Choices) == 0 || payload.Choices[0].Delta.Content == "" {
				continue
			}

			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case chunks <- payload.Choices[0].Delta.Content:
			}
		}

		if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
			errCh <- fmt.Errorf("failed to read OpenRouter stream: %w", err)
			return
		}
		usageCh <- usage
	}()

	return chunks, usageCh, errCh
}

func (c *client) callAPI(ctx context.Context, messages []Message, model *string, stream bool, temperature float64) (*http.Response, error) {
	if c.cfg.APIKey == "" {
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
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.HTTPReferer != "" {
		req.Header.Set("HTTP-Referer", c.cfg.HTTPReferer)
	}
	if c.cfg.Title != "" {
		req.Header.Set("X-Title", c.cfg.Title)
	}

	resp, err := c.httpClient.Do(req)
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
