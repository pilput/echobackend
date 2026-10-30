package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/service"
	"echobackend/pkg/response"
	"echobackend/pkg/validator"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

// mockChatService embeds the interface so any method a test does not stub
// panics, which doubles as a "service must not be called" assertion.
type mockChatService struct {
	service.ChatConversationService
	createStreamingMessageFn func(ctx context.Context, userID, conversationID string, req *dto.CreateChatMessageRequest) (*dto.ChatStreamResult, <-chan string, <-chan dto.ChatMessageResponse, <-chan error, error)
}

func (m *mockChatService) CreateStreamingMessage(ctx context.Context, userID, conversationID string, req *dto.CreateChatMessageRequest) (*dto.ChatStreamResult, <-chan string, <-chan dto.ChatMessageResponse, <-chan error, error) {
	return m.createStreamingMessageFn(ctx, userID, conversationID, req)
}

func registerChatRoutes(e *echo.Echo, h *ChatConversationHandler) {
	e.GET("/api/chat/conversations/:id", h.GetConversation)
	e.PUT("/api/chat/conversations/:id", h.UpdateConversation)
	e.DELETE("/api/chat/conversations/:id", h.DeleteConversation)
	e.POST("/api/chat/conversations/:conversationId/messages", h.CreateMessage)
	e.POST("/api/chat/conversations/:conversationId/messages/stream", h.CreateMessageStream)
	e.GET("/api/chat/conversations/:conversationId/messages", h.GetMessages)
	e.GET("/api/chat/messages/:messageId", h.GetMessage)
	e.DELETE("/api/chat/messages/:messageId", h.DeleteMessage)
}

func serveChatRoute(t *testing.T, svc service.ChatConversationService, method, target, body string) (int, string, response.APIResponse) {
	t.Helper()
	h := NewChatConversationHandler(svc)
	rec := serveAuthRoute(t, func(e *echo.Echo, _ *AuthHandler) { registerChatRoutes(e, h) }, nil, method, target, body, "user-1")
	raw := rec.Body.String()
	var out response.APIResponse
	_ = json.Unmarshal([]byte(raw), &out) // SSE bodies are not JSON; callers check raw then
	return rec.Code, raw, out
}

const validChatID = "0190a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b"

func TestChatHandlerRejectsMalformedIDs(t *testing.T) {
	cases := []struct {
		method, target, body, want string
	}{
		{http.MethodGet, "/api/chat/conversations/abc", "", MsgInvalidConversationID},
		{http.MethodPut, "/api/chat/conversations/abc", `{"title":"x"}`, MsgInvalidConversationID},
		{http.MethodDelete, "/api/chat/conversations/abc", "", MsgInvalidConversationID},
		{http.MethodPost, "/api/chat/conversations/abc/messages", `{"content":"hi"}`, MsgInvalidConversationID},
		{http.MethodPost, "/api/chat/conversations/abc/messages/stream", `{"content":"hi"}`, MsgInvalidConversationID},
		{http.MethodGet, "/api/chat/conversations/abc/messages", "", MsgInvalidConversationID},
		{http.MethodGet, "/api/chat/messages/abc", "", MsgInvalidMessageID},
		{http.MethodDelete, "/api/chat/messages/abc", "", MsgInvalidMessageID},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.target, func(t *testing.T) {
			status, _, out := serveChatRoute(t, &mockChatService{}, tc.method, tc.target, tc.body)
			if status != http.StatusBadRequest || out.Message != tc.want || out.Success {
				t.Fatalf("got %d %q, want 400 %q", status, out.Message, tc.want)
			}
		})
	}
}

// streamWith returns service channels that deliver chunks, then err (if any)
// or the completed message, then close — like the real producer goroutine.
func streamWith(chunks []string, streamErr error, done *dto.ChatMessageResponse) (<-chan string, <-chan dto.ChatMessageResponse, <-chan error) {
	chunkCh := make(chan string, len(chunks))
	for _, c := range chunks {
		chunkCh <- c
	}
	close(chunkCh)
	complete := make(chan dto.ChatMessageResponse, 1)
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

func TestChatMessageStreamEndsWithDoneAfterError(t *testing.T) {
	svc := &mockChatService{
		createStreamingMessageFn: func(ctx context.Context, userID, conversationID string, req *dto.CreateChatMessageRequest) (*dto.ChatStreamResult, <-chan string, <-chan dto.ChatMessageResponse, <-chan error, error) {
			chunks, complete, errCh := streamWith([]string{"Hel"}, apperrors.ErrChatAIResponseSaveFailed, nil)
			return &dto.ChatStreamResult{UserMessage: &dto.ChatMessageResponse{ID: "m-1", Role: "user"}}, chunks, complete, errCh, nil
		},
	}
	status, raw, _ := serveChatRoute(t, svc, http.MethodPost, "/api/chat/conversations/"+validChatID+"/messages/stream", `{"content":"hi"}`)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", status, raw)
	}
	wantOrder := []string{`"type":"user_message"`, `"type":"ai_chunk"`, `{"data":"Failed to save AI response","type":"error"}`, "data: [DONE]\n\n"}
	pos := 0
	for _, w := range wantOrder {
		i := strings.Index(raw[pos:], w)
		if i < 0 {
			t.Fatalf("missing %q after offset %d in stream:\n%s", w, pos, raw)
		}
		pos += i + len(w)
	}
	if !strings.HasSuffix(raw, "data: [DONE]\n\n") || strings.Contains(raw, "ai_complete") {
		t.Fatalf("stream must end with [DONE] and carry no ai_complete:\n%s", raw)
	}
}

func TestChatMessageStreamEndsWithDoneAfterComplete(t *testing.T) {
	svc := &mockChatService{
		createStreamingMessageFn: func(ctx context.Context, userID, conversationID string, req *dto.CreateChatMessageRequest) (*dto.ChatStreamResult, <-chan string, <-chan dto.ChatMessageResponse, <-chan error, error) {
			chunks, complete, errCh := streamWith([]string{"Hi"}, nil, &dto.ChatMessageResponse{ID: "m-2", Role: "assistant", Content: "Hi"})
			return &dto.ChatStreamResult{UserMessage: &dto.ChatMessageResponse{ID: "m-1", Role: "user"}}, chunks, complete, errCh, nil
		},
	}
	status, raw, _ := serveChatRoute(t, svc, http.MethodPost, "/api/chat/conversations/"+validChatID+"/messages/stream", `{"content":"hi"}`)
	if status != http.StatusOK || !strings.Contains(raw, `"type":"ai_complete"`) || !strings.HasSuffix(raw, "data: [DONE]\n\n") {
		t.Fatalf("got %d:\n%s", status, raw)
	}
}

func TestChatMessageStreamFallbackIs201(t *testing.T) {
	svc := &mockChatService{
		createStreamingMessageFn: func(ctx context.Context, userID, conversationID string, req *dto.CreateChatMessageRequest) (*dto.ChatStreamResult, <-chan string, <-chan dto.ChatMessageResponse, <-chan error, error) {
			return &dto.ChatStreamResult{UserMessage: &dto.ChatMessageResponse{ID: "m-1", Role: "user"}}, nil, nil, nil, nil
		},
	}
	status, _, out := serveChatRoute(t, svc, http.MethodPost, "/api/chat/conversations/"+validChatID+"/messages/stream", `{"content":"hi"}`)
	if status != http.StatusCreated || out.Message != "Message created successfully" {
		t.Fatalf("got %d %q, want 201 fallback", status, out.Message)
	}
}

func TestStreamErrorMessage(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{apperrors.ErrChatAIResponseSaveFailed, "Failed to save AI response"},
		{errors.New(`OpenRouter API error: 429 {"error":{"message":"Rate limited"}}`), "Rate limited"},
		{errors.New("OpenRouter returned an empty response"), "OpenRouter returned an empty response"},
	}
	for _, tc := range cases {
		if got := streamErrorMessage(tc.err); got != tc.want {
			t.Fatalf("streamErrorMessage(%v) = %q, want %q", tc.err, got, tc.want)
		}
	}
}

func TestChatMessageStreamRejectsInvalidBody(t *testing.T) {
	status, _, out := serveChatRoute(t, &mockChatService{}, http.MethodPost, "/api/chat/conversations/"+validChatID+"/messages/stream", `{}`)
	if status != http.StatusUnprocessableEntity || out.Message != "Validation failed" {
		t.Fatalf("got %d %q, want 422 (and no service call)", status, out.Message)
	}
}

// TestChatMessageStreamOutlivesServerTimeouts runs the stream on a real server
// whose Read/WriteTimeout are far shorter than the stream: the handler must
// lift the read deadline and push the write deadline per frame, or the client
// never sees ai_complete and [DONE] (and the request ctx gets cancelled).
func TestChatMessageStreamOutlivesServerTimeouts(t *testing.T) {
	const serverTimeout = 150 * time.Millisecond
	svc := &mockChatService{
		createStreamingMessageFn: func(ctx context.Context, userID, conversationID string, req *dto.CreateChatMessageRequest) (*dto.ChatStreamResult, <-chan string, <-chan dto.ChatMessageResponse, <-chan error, error) {
			chunks := make(chan string)
			complete := make(chan dto.ChatMessageResponse, 1)
			errCh := make(chan error, 1)
			go func() {
				defer close(chunks)
				defer close(complete)
				defer close(errCh)
				for _, c := range []string{"a", "b", "c"} {
					select {
					case <-ctx.Done():
						errCh <- ctx.Err()
						return
					case <-time.After(2 * serverTimeout):
					}
					select {
					case <-ctx.Done():
						errCh <- ctx.Err()
						return
					case chunks <- c:
					}
				}
				complete <- dto.ChatMessageResponse{ID: "m-2", Role: "assistant", Content: "abc"}
			}()
			return &dto.ChatStreamResult{UserMessage: &dto.ChatMessageResponse{ID: "m-1", Role: "user"}}, chunks, complete, errCh, nil
		},
	}

	e := echo.New()
	e.Validator = validator.NewValidator()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("user", jwt.MapClaims{"user_id": "user-1"})
			return next(c)
		}
	})
	registerChatRoutes(e, NewChatConversationHandler(svc))
	srv := httptest.NewUnstartedServer(e)
	srv.Config.ReadTimeout = serverTimeout
	srv.Config.WriteTimeout = serverTimeout
	srv.Start()
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL+"/api/chat/conversations/"+validChatID+"/messages/stream", strings.NewReader(`{"content":"hi"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST stream: %v", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read stream: %v (got so far: %s)", err, body)
	}
	raw := string(body)
	if !strings.Contains(raw, `"type":"ai_complete"`) || !strings.HasSuffix(raw, "data: [DONE]\n\n") {
		t.Fatalf("stream cut short by server timeouts:\n%s", raw)
	}
}
