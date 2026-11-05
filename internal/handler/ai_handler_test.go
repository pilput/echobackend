package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"echobackend/internal/dto"
	"echobackend/pkg/response"

	"github.com/labstack/echo/v5"
)

// mockArticleService: a nil generateFn panics on call, doubling as a
// "service must not be called" assertion.
type mockArticleService struct {
	available  bool
	generateFn func(ctx context.Context, req *dto.GenerateArticleRequest) (<-chan string, <-chan dto.GeneratedArticle, <-chan error, error)
}

func (m *mockArticleService) Available() bool { return m.available }

func (m *mockArticleService) GenerateArticleStream(ctx context.Context, req *dto.GenerateArticleRequest) (<-chan string, <-chan dto.GeneratedArticle, <-chan error, error) {
	return m.generateFn(ctx, req)
}

func articleStreamWith(chunks []string, streamErr error, done *dto.GeneratedArticle) (<-chan string, <-chan dto.GeneratedArticle, <-chan error) {
	chunkCh := make(chan string, len(chunks))
	for _, c := range chunks {
		chunkCh <- c
	}
	close(chunkCh)
	complete := make(chan dto.GeneratedArticle, 1)
	errCh := make(chan error, 1)
	if streamErr != nil {
		errCh <- streamErr
	} else if done != nil {
		complete <- *done
	}
	close(complete)
	close(errCh)
	return chunkCh, complete, errCh
}

const generateArticlePath = "/api/ai/generate-article"

func serveAIRoute(t *testing.T, svc *mockArticleService, body, actorID string) (int, string, response.APIResponse) {
	t.Helper()
	h := NewAIHandler(svc)
	rec := serveAuthRoute(t, func(e *echo.Echo, _ *AuthHandler) {
		e.POST(generateArticlePath, h.GenerateArticle)
	}, nil, http.MethodPost, generateArticlePath, body, actorID)
	raw := rec.Body.String()
	var out response.APIResponse
	_ = json.Unmarshal([]byte(raw), &out)
	return rec.Code, raw, out
}

func assertInOrder(t *testing.T, raw string, want []string) {
	t.Helper()
	pos := 0
	for _, w := range want {
		i := strings.Index(raw[pos:], w)
		if i < 0 {
			t.Fatalf("missing %q after offset %d in stream:\n%s", w, pos, raw)
		}
		pos += i + len(w)
	}
}

func TestGenerateArticleStreamsChunksThenComplete(t *testing.T) {
	var gotReq dto.GenerateArticleRequest
	svc := &mockArticleService{
		available: true,
		generateFn: func(ctx context.Context, req *dto.GenerateArticleRequest) (<-chan string, <-chan dto.GeneratedArticle, <-chan error, error) {
			gotReq = *req
			chunks, complete, errCh := articleStreamWith([]string{"<h1>A</h1>", "<p>b</p>"}, nil, &dto.GeneratedArticle{Content: "<h1>A</h1><p>b</p>", Model: "test-model"})
			return chunks, complete, errCh, nil
		},
	}
	status, raw, _ := serveAIRoute(t, svc, `{"prompt":"  Go generics  ","tone":"casual","title":"  T  ","model":"ignored"}`, "user-1")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", status, raw)
	}
	want := "data: {\"data\":\"\\u003ch1\\u003eA\\u003c/h1\\u003e\",\"type\":\"ai_chunk\"}\n\n" +
		"data: {\"data\":\"\\u003cp\\u003eb\\u003c/p\\u003e\",\"type\":\"ai_chunk\"}\n\n" +
		"data: {\"data\":{\"content\":\"\\u003ch1\\u003eA\\u003c/h1\\u003e\\u003cp\\u003eb\\u003c/p\\u003e\",\"model\":\"test-model\"},\"type\":\"ai_complete\"}\n\n" +
		"data: [DONE]\n\n"
	if raw != want {
		t.Fatalf("stream mismatch:\ngot:\n%s\nwant:\n%s", raw, want)
	}
	if gotReq.Prompt != "Go generics" || gotReq.Title != "T" || gotReq.Tone != "casual" {
		t.Fatalf("request not trimmed/bound: %+v", gotReq)
	}
}

func TestGenerateArticleStreamEndsWithDoneAfterError(t *testing.T) {
	svc := &mockArticleService{
		available: true,
		generateFn: func(ctx context.Context, req *dto.GenerateArticleRequest) (<-chan string, <-chan dto.GeneratedArticle, <-chan error, error) {
			chunks, complete, errCh := articleStreamWith([]string{"x"}, errors.New(`OpenRouter API error: 429 {"error":{"message":"Rate limited"}}`), nil)
			return chunks, complete, errCh, nil
		},
	}
	status, raw, _ := serveAIRoute(t, svc, `{"prompt":"abc"}`, "user-1")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", status, raw)
	}
	assertInOrder(t, raw, []string{`"type":"ai_chunk"`, `{"data":"Rate limited","type":"error"}`, "data: [DONE]\n\n"})
	if !strings.HasSuffix(raw, "data: [DONE]\n\n") || strings.Contains(raw, "ai_complete") {
		t.Fatalf("stream must end with [DONE] and carry no ai_complete:\n%s", raw)
	}
}

func TestGenerateArticleUnavailableIs503(t *testing.T) {
	status, _, out := serveAIRoute(t, &mockArticleService{available: false}, `{"prompt":"abc"}`, "user-1")
	if status != http.StatusServiceUnavailable || out.Success || out.Message != "AI article generation is not available" {
		t.Fatalf("got %d %+v, want 503 unavailable", status, out)
	}
}

func TestGenerateArticleRejectsInvalidBody(t *testing.T) {
	cases := []struct{ name, body string }{
		{"missing prompt", `{}`},
		{"short prompt after trim", `{"prompt":"  ab  "}`},
		{"long prompt", `{"prompt":"` + strings.Repeat("a", 2001) + `"}`},
		{"bad tone", `{"prompt":"abc","tone":"angry"}`},
		{"bad length", `{"prompt":"abc","length":"huge"}`},
		{"long title", `{"prompt":"abc","title":"` + strings.Repeat("t", 201) + `"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, _, out := serveAIRoute(t, &mockArticleService{available: true}, tc.body, "user-1")
			if status != http.StatusUnprocessableEntity || out.Message != "Validation failed" {
				t.Fatalf("got %d %q, want 422 (and no service call)", status, out.Message)
			}
		})
	}
}

func TestGenerateArticleRejectsMalformedJSON(t *testing.T) {
	status, _, out := serveAIRoute(t, &mockArticleService{available: true}, `{"prompt":`, "user-1")
	if status != http.StatusBadRequest || out.Success {
		t.Fatalf("got %d %+v, want 400", status, out)
	}
}

func TestGenerateArticleRequiresUser(t *testing.T) {
	status, _, out := serveAIRoute(t, &mockArticleService{available: true}, `{"prompt":"abc"}`, "")
	if status != http.StatusUnauthorized || out.Message != "User not authenticated" {
		t.Fatalf("got %d %q, want 401", status, out.Message)
	}
}
