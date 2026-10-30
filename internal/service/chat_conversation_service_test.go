package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"echobackend/config"
	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/model"
	"echobackend/internal/platform/openrouter"
)

type mockChatConversationRepo struct {
	getConversationByIDFn            func(ctx context.Context, id string) (*model.ChatConversation, error)
	createMessageFn                  func(ctx context.Context, message *model.ChatMessage) (*model.ChatMessage, error)
	updateConversationFn             func(ctx context.Context, id string, updates map[string]any) (*model.ChatConversation, error)
	getMessagesByConversationIDAscFn func(ctx context.Context, conversationID, userID string) ([]*model.ChatMessage, error)
	getUserConversationsFn           func(ctx context.Context, userID string, offset int, limit int) ([]*model.ChatConversation, int64, error)
	getMessageByIDFn                 func(ctx context.Context, id, userID string) (*model.ChatMessage, error)
	deleteMessageFn                  func(ctx context.Context, id, userID string) error
}

func (m *mockChatConversationRepo) CreateConversation(ctx context.Context, conversation *model.ChatConversation) (*model.ChatConversation, error) {
	return conversation, nil
}

func (m *mockChatConversationRepo) GetConversationByID(ctx context.Context, id string) (*model.ChatConversation, error) {
	if m.getConversationByIDFn != nil {
		return m.getConversationByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockChatConversationRepo) GetConversationsByUserID(ctx context.Context, userID string, offset int, limit int) ([]*model.ChatConversation, int64, error) {
	return nil, 0, nil
}

func (m *mockChatConversationRepo) UpdateConversation(ctx context.Context, id string, updates map[string]any) (*model.ChatConversation, error) {
	if m.updateConversationFn != nil {
		return m.updateConversationFn(ctx, id, updates)
	}
	return &model.ChatConversation{ID: id}, nil
}

func (m *mockChatConversationRepo) DeleteConversation(ctx context.Context, id string) error {
	return nil
}

func (m *mockChatConversationRepo) GetUserConversations(ctx context.Context, userID string, offset int, limit int) ([]*model.ChatConversation, int64, error) {
	if m.getUserConversationsFn != nil {
		return m.getUserConversationsFn(ctx, userID, offset, limit)
	}
	return nil, 0, nil
}

func (m *mockChatConversationRepo) CreateMessage(ctx context.Context, message *model.ChatMessage) (*model.ChatMessage, error) {
	if m.createMessageFn != nil {
		return m.createMessageFn(ctx, message)
	}
	return message, nil
}

func (m *mockChatConversationRepo) GetMessagesByConversationID(ctx context.Context, conversationID, userID string) ([]*model.ChatMessage, error) {
	return nil, nil
}

func (m *mockChatConversationRepo) GetMessagesByConversationIDAsc(ctx context.Context, conversationID, userID string) ([]*model.ChatMessage, error) {
	if m.getMessagesByConversationIDAscFn != nil {
		return m.getMessagesByConversationIDAscFn(ctx, conversationID, userID)
	}
	return nil, nil
}

func (m *mockChatConversationRepo) GetMessageByID(ctx context.Context, id, userID string) (*model.ChatMessage, error) {
	if m.getMessageByIDFn != nil {
		return m.getMessageByIDFn(ctx, id, userID)
	}
	return nil, nil
}

func (m *mockChatConversationRepo) DeleteMessage(ctx context.Context, id, userID string) error {
	if m.deleteMessageFn != nil {
		return m.deleteMessageFn(ctx, id, userID)
	}
	return nil
}

type mockOpenRouterService struct {
	generateStreamFn   func(ctx context.Context, messages []openrouter.Message, model *string, temperature float64) (<-chan string, <-chan openrouter.Usage, <-chan error)
	generateResponseFn func(ctx context.Context, messages []openrouter.Message, model *string, temperature float64) (*openrouter.Response, error)
}

func (m *mockOpenRouterService) GenerateResponse(ctx context.Context, messages []openrouter.Message, model *string, temperature float64) (*openrouter.Response, error) {
	if m.generateResponseFn != nil {
		return m.generateResponseFn(ctx, messages, model, temperature)
	}
	return nil, nil
}

func (m *mockOpenRouterService) GenerateStream(ctx context.Context, messages []openrouter.Message, model *string, temperature float64) (<-chan string, <-chan openrouter.Usage, <-chan error) {
	if m.generateStreamFn != nil {
		return m.generateStreamFn(ctx, messages, model, temperature)
	}
	return nil, nil, nil
}

func TestBuildConversationTitle(t *testing.T) {
	t.Parallel()

	ptr := func(s string) *string { return &s }

	tests := []struct {
		name    string
		title   *string
		content string
		want    string
	}{
		{
			name:    "explicit title respected",
			title:   ptr("  Laporan Q1  "),
			content: "abaikan ini",
			want:    "Laporan Q1",
		},
		{
			name:    "empty content falls back",
			title:   nil,
			content: "   ",
			want:    "New conversation",
		},
		{
			name:    "strips leading filler words",
			title:   nil,
			content: "tolong buatkan laporan keuangan bulan januari untuk presentasi besok",
			want:    "laporan keuangan bulan januari untuk presentasi besok",
		},
		{
			name:    "takes first sentence",
			title:   nil,
			content: "Apa itu inflasi? Jelaskan dampaknya ke pasar saham Indonesia secara detail",
			want:    "inflasi",
		},
		{
			name:    "truncates at word boundary",
			title:   nil,
			content: "analisis perbandingan saham bank BCA BRI Mandiri BNI BTN CIMB Danamon Permata OCBC tahun ini",
			want:    "analisis perbandingan saham bank BCA BRI Mandiri BNI",
		},
		{
			name:    "never splits UTF-8 rune",
			title:   nil,
			content: "ceritakan tentang café crème brûlée naïve façade di kota Zürich yang indah sekali dan menawan hati",
			want:    "ceritakan tentang café crème brûlée naïve façade di",
		},
		{
			name:    "strips markdown and list prefix",
			title:   nil,
			content: "### 1. Jelaskan strategi investasi untuk pemula yang baru mulai",
			want:    "strategi investasi untuk pemula yang baru mulai",
		},
		{
			name:    "strips trailing filler",
			title:   nil,
			content: "jelaskan bedanya saham dan obligasi untuk pemula ya",
			want:    "bedanya saham dan obligasi untuk pemula",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := buildConversationTitle(tt.title, tt.content); got != tt.want {
				t.Fatalf("buildConversationTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildConversationTitle_ExplicitLongTitleTruncatedToDBLimit(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("a", 300)
	got := buildConversationTitle(&long, "fallback")
	if gotLen := len([]rune(got)); gotLen != maxExplicitTitleRunes {
		t.Fatalf("explicit title runes = %d, want %d", gotLen, maxExplicitTitleRunes)
	}
}

func TestCreateStreamingMessage_ForwardsAllChunksToClient(t *testing.T) {
	t.Parallel()

	const (
		userID         = "user-1"
		conversationID = "conv-1"
	)

	repo := &mockChatConversationRepo{
		getConversationByIDFn: func(ctx context.Context, id string) (*model.ChatConversation, error) {
			return &model.ChatConversation{
				ID:     conversationID,
				UserID: userID,
				Title:  "Existing conversation",
			}, nil
		},
		createMessageFn: func(ctx context.Context, message *model.ChatMessage) (*model.ChatMessage, error) {
			if message.Role == "assistant" {
				message.ID = "assistant-1"
				return message, nil
			}
			message.ID = "user-1"
			return message, nil
		},
		getMessagesByConversationIDAscFn: func(ctx context.Context, conversationID, userID string) ([]*model.ChatMessage, error) {
			return []*model.ChatMessage{{
				ID:             "user-1",
				ConversationID: conversationID,
				UserID:         userID,
				Role:           "user",
				Content:        "Hello",
			}}, nil
		},
	}

	openRouter := &mockOpenRouterService{
		generateStreamFn: func(ctx context.Context, messages []openrouter.Message, model *string, temperature float64) (<-chan string, <-chan openrouter.Usage, <-chan error) {
			chunks := make(chan string, 2)
			usageCh := make(chan openrouter.Usage, 1)
			errCh := make(chan error, 1)

			chunks <- "Hello "
			chunks <- "world"
			close(chunks)
			usageCh <- openrouter.Usage{PromptTokens: 1, CompletionTokens: 2, TotalTokens: 3}
			close(usageCh)
			close(errCh)

			return chunks, usageCh, errCh
		},
	}

	svc := NewChatConversationService(repo, openRouter, aiTestConfig())

	result, chunks, complete, errCh, err := svc.CreateStreamingMessage(context.Background(), userID, conversationID, &dto.CreateChatMessageRequest{
		Content: "Hello",
	})
	if err != nil {
		t.Fatalf("CreateStreamingMessage() error = %v", err)
	}
	if result == nil || result.UserMessage == nil {
		t.Fatal("expected user message in stream result")
	}

	var received []string
	for chunk := range chunks {
		received = append(received, chunk)
	}

	if len(received) != 2 || received[0] != "Hello " || received[1] != "world" {
		t.Fatalf("client chunks = %#v, want [Hello , world]", received)
	}

	if err, ok := <-errCh; ok && err != nil {
		t.Fatalf("unexpected stream error: %v", err)
	}

	msg, ok := <-complete
	if !ok {
		t.Fatal("expected completed assistant message")
	}
	if msg.Content != "Hello world" {
		t.Fatalf("assistant content = %q, want %q", msg.Content, "Hello world")
	}
}

// aiTestConfig is a config under which AI replies are available.
func aiTestConfig() *config.Config {
	return &config.Config{
		OpenRouter: config.OpenRouterConfig{APIKey: "test-key", DefaultModel: "test-model"},
	}
}

func ownedConversationRepo(userID, conversationID, title string) *mockChatConversationRepo {
	return &mockChatConversationRepo{
		getConversationByIDFn: func(ctx context.Context, id string) (*model.ChatConversation, error) {
			return &model.ChatConversation{ID: conversationID, UserID: userID, Title: title}, nil
		},
		getMessagesByConversationIDAscFn: func(ctx context.Context, conversationID, userID string) ([]*model.ChatMessage, error) {
			return []*model.ChatMessage{{ID: "m-1", ConversationID: conversationID, UserID: userID, Role: "user", Content: "Hello"}}, nil
		},
	}
}

func TestAIAvailable(t *testing.T) {
	t.Parallel()

	ptr := func(s string) *string { return &s }
	tests := []struct {
		name   string
		apiKey string
		def    string
		model  *string
		want   bool
	}{
		{name: "key and default model", apiKey: "k", def: "openrouter/free", want: true},
		{name: "blank key", apiKey: "   ", def: "openrouter/free", want: false},
		{name: "empty key", apiKey: "", def: "openrouter/free", model: ptr("m"), want: false},
		{name: "no model anywhere", apiKey: "k", def: " ", model: ptr("  "), want: false},
		{name: "request model only", apiKey: "k", def: "", model: ptr("m"), want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := NewChatConversationService(&mockChatConversationRepo{}, &mockOpenRouterService{}, &config.Config{
				OpenRouter: config.OpenRouterConfig{APIKey: tt.apiKey, DefaultModel: tt.def},
			}).(*chatConversationService)
			if got := svc.aiAvailable(tt.model); got != tt.want {
				t.Fatalf("aiAvailable() = %v, want %v", got, tt.want)
			}
		})
	}

	nilClient := NewChatConversationService(&mockChatConversationRepo{}, nil, aiTestConfig()).(*chatConversationService)
	if nilClient.aiAvailable(nil) {
		t.Fatal("aiAvailable() must be false without a client")
	}
}

func TestCreateMessage_AIUnavailableStoresUserMessageOnly(t *testing.T) {
	t.Parallel()

	repo := ownedConversationRepo("user-1", "conv-1", "New conversation")
	var updates []map[string]any
	repo.updateConversationFn = func(ctx context.Context, id string, u map[string]any) (*model.ChatConversation, error) {
		updates = append(updates, u)
		return &model.ChatConversation{ID: id}, nil
	}
	openRouter := &mockOpenRouterService{
		generateResponseFn: func(ctx context.Context, messages []openrouter.Message, model *string, temperature float64) (*openrouter.Response, error) {
			t.Fatal("OpenRouter must not be called without an API key")
			return nil, nil
		},
	}
	svc := NewChatConversationService(repo, openRouter, &config.Config{
		OpenRouter: config.OpenRouterConfig{APIKey: "  ", DefaultModel: "openrouter/free"},
	})

	got, err := svc.CreateMessage(context.Background(), "user-1", "conv-1", &dto.CreateChatMessageRequest{Content: "Hello"})
	if err != nil {
		t.Fatalf("CreateMessage() error = %v", err)
	}
	if len(got) != 1 || got[0].Role != "user" {
		t.Fatalf("CreateMessage() = %#v, want only the user message", got)
	}
	if len(updates) != 1 {
		t.Fatalf("conversation updates = %#v, want only the updated_at touch", updates)
	}
	if _, ok := updates[0]["title"]; ok {
		t.Fatalf("conversation must not be auto-titled when AI is unavailable: %#v", updates[0])
	}
}

func TestCreateStreamingMessage_AIUnavailableReturnsFallback(t *testing.T) {
	t.Parallel()

	repo := ownedConversationRepo("user-1", "conv-1", "New conversation")
	openRouter := &mockOpenRouterService{
		generateStreamFn: func(ctx context.Context, messages []openrouter.Message, model *string, temperature float64) (<-chan string, <-chan openrouter.Usage, <-chan error) {
			t.Fatal("OpenRouter must not be called without an API key")
			return nil, nil, nil
		},
	}
	svc := NewChatConversationService(repo, openRouter, &config.Config{
		OpenRouter: config.OpenRouterConfig{DefaultModel: "openrouter/free"},
	})

	result, chunks, _, _, err := svc.CreateStreamingMessage(context.Background(), "user-1", "conv-1", &dto.CreateChatMessageRequest{Content: "Hello"})
	if err != nil {
		t.Fatalf("CreateStreamingMessage() error = %v", err)
	}
	if chunks != nil {
		t.Fatal("chunks must be nil so the handler answers 201 JSON instead of SSE")
	}
	if result == nil || result.UserMessage == nil {
		t.Fatal("expected the stored user message")
	}
}

func TestCreateStreamingMessage_SaveFailureSendsGenericError(t *testing.T) {
	t.Parallel()

	repo := ownedConversationRepo("user-1", "conv-1", "Existing conversation")
	repo.createMessageFn = func(ctx context.Context, message *model.ChatMessage) (*model.ChatMessage, error) {
		if message.Role == "assistant" {
			return nil, errors.New(`pq: relation "custom.chat_messages" does not exist`)
		}
		return message, nil
	}
	openRouter := &mockOpenRouterService{
		generateStreamFn: func(ctx context.Context, messages []openrouter.Message, model *string, temperature float64) (<-chan string, <-chan openrouter.Usage, <-chan error) {
			chunks := make(chan string, 1)
			usageCh := make(chan openrouter.Usage)
			errCh := make(chan error)
			chunks <- "Hi"
			close(chunks)
			close(usageCh)
			close(errCh)
			return chunks, usageCh, errCh
		},
	}
	svc := NewChatConversationService(repo, openRouter, aiTestConfig())

	_, chunks, complete, errCh, err := svc.CreateStreamingMessage(context.Background(), "user-1", "conv-1", &dto.CreateChatMessageRequest{Content: "Hello"})
	if err != nil {
		t.Fatalf("CreateStreamingMessage() error = %v", err)
	}
	drained := 0
	for range chunks {
		drained++
	}
	if drained != 1 {
		t.Fatalf("chunks = %d, want 1", drained)
	}
	streamErr := <-errCh
	if !errors.Is(streamErr, apperrors.ErrChatAIResponseSaveFailed) {
		t.Fatalf("stream error = %v, want ErrChatAIResponseSaveFailed", streamErr)
	}
	if _, ok := <-complete; ok {
		t.Fatal("no ai_complete expected after a save failure")
	}
}

func TestGetUserConversations_EmptyIsNonNilSlice(t *testing.T) {
	t.Parallel()

	svc := NewChatConversationService(&mockChatConversationRepo{}, nil, nil)
	got, total, err := svc.GetUserConversations(context.Background(), "user-1", 0, 10)
	if err != nil {
		t.Fatalf("GetUserConversations() error = %v", err)
	}
	if got == nil || len(got) != 0 || total != 0 {
		t.Fatalf("GetUserConversations() = %#v, %d; want empty non-nil slice", got, total)
	}
}

func TestUpdateConversation_WhitespaceTitleIsIgnored(t *testing.T) {
	t.Parallel()

	repo := ownedConversationRepo("user-1", "conv-1", "Old title")
	var got map[string]any
	repo.updateConversationFn = func(ctx context.Context, id string, u map[string]any) (*model.ChatConversation, error) {
		got = u
		return &model.ChatConversation{ID: id, Title: "Old title"}, nil
	}
	svc := NewChatConversationService(repo, nil, nil)

	if _, err := svc.UpdateConversation(context.Background(), "conv-1", "user-1", &dto.UpdateChatConversationRequest{Title: "   "}); err != nil {
		t.Fatalf("UpdateConversation() error = %v", err)
	}
	if _, ok := got["title"]; ok {
		t.Fatalf("whitespace-only title must not be stored: %#v", got)
	}

	if _, err := svc.UpdateConversation(context.Background(), "conv-1", "user-1", &dto.UpdateChatConversationRequest{Title: " New "}); err != nil {
		t.Fatalf("UpdateConversation() error = %v", err)
	}
	if got["title"] != " New " {
		t.Fatalf("title = %#v, want the untrimmed %q", got["title"], " New ")
	}
}

func TestGetMessage_ChecksParentConversation(t *testing.T) {
	t.Parallel()

	message := &model.ChatMessage{ID: "m-1", ConversationID: "conv-1", UserID: "user-1"}
	tests := []struct {
		name    string
		convErr error
		owner   string
		wantErr error
	}{
		{name: "soft-deleted conversation", convErr: apperrors.ErrChatConversationNotFound, wantErr: apperrors.ErrChatConversationNotFound},
		{name: "other owner", owner: "user-2", wantErr: apperrors.ErrConversationNotOwned},
		{name: "owned", owner: "user-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := &mockChatConversationRepo{
				getMessageByIDFn: func(ctx context.Context, id, userID string) (*model.ChatMessage, error) {
					return message, nil
				},
				getConversationByIDFn: func(ctx context.Context, id string) (*model.ChatConversation, error) {
					if id != "conv-1" {
						t.Fatalf("conversation lookup id = %q, want conv-1", id)
					}
					if tt.convErr != nil {
						return nil, tt.convErr
					}
					return &model.ChatConversation{ID: id, UserID: tt.owner}, nil
				},
			}
			svc := NewChatConversationService(repo, nil, nil)
			_, err := svc.GetMessage(context.Background(), "m-1", "user-1")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("GetMessage() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeleteMessage_ChecksConversationAndBumpsUpdatedAt(t *testing.T) {
	t.Parallel()

	message := &model.ChatMessage{ID: "m-1", ConversationID: "conv-1", UserID: "user-1"}
	deleted := false
	var touched map[string]any
	repo := &mockChatConversationRepo{
		getMessageByIDFn: func(ctx context.Context, id, userID string) (*model.ChatMessage, error) {
			return message, nil
		},
		getConversationByIDFn: func(ctx context.Context, id string) (*model.ChatConversation, error) {
			return &model.ChatConversation{ID: id, UserID: "user-1"}, nil
		},
		deleteMessageFn: func(ctx context.Context, id, userID string) error {
			deleted = true
			return nil
		},
		updateConversationFn: func(ctx context.Context, id string, u map[string]any) (*model.ChatConversation, error) {
			if id != "conv-1" {
				t.Fatalf("touched conversation %q, want conv-1", id)
			}
			touched = u
			return &model.ChatConversation{ID: id}, nil
		},
	}
	svc := NewChatConversationService(repo, nil, nil)

	got, err := svc.DeleteMessage(context.Background(), "m-1", "user-1")
	if err != nil {
		t.Fatalf("DeleteMessage() error = %v", err)
	}
	if !deleted || got == nil || got.ID != "m-1" {
		t.Fatalf("DeleteMessage() = %#v (deleted=%v), want the deleted message echoed", got, deleted)
	}
	if _, ok := touched["updated_at"]; !ok {
		t.Fatalf("conversation updated_at not bumped: %#v", touched)
	}

	// A message in a soft-deleted conversation is not deletable.
	repo.getConversationByIDFn = func(ctx context.Context, id string) (*model.ChatConversation, error) {
		return nil, apperrors.ErrChatConversationNotFound
	}
	deleted = false
	if _, err := svc.DeleteMessage(context.Background(), "m-1", "user-1"); !errors.Is(err, apperrors.ErrChatConversationNotFound) {
		t.Fatalf("DeleteMessage() error = %v, want ErrChatConversationNotFound", err)
	}
	if deleted {
		t.Fatal("message of a soft-deleted conversation must not be deleted")
	}
}
