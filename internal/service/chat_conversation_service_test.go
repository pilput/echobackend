package service

import (
	"context"
	"strings"
	"testing"

	"echobackend/config"
	"echobackend/internal/dto"
	"echobackend/internal/model"
	"echobackend/internal/platform/openrouter"
)

type mockChatConversationRepo struct {
	getConversationByIDFn            func(ctx context.Context, id string) (*model.ChatConversation, error)
	createMessageFn                  func(ctx context.Context, message *model.ChatMessage) (*model.ChatMessage, error)
	updateConversationFn             func(ctx context.Context, id string, updates map[string]any) (*model.ChatConversation, error)
	getMessagesByConversationIDAscFn func(ctx context.Context, conversationID, userID string) ([]*model.ChatMessage, error)
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
	return nil, nil
}

func (m *mockChatConversationRepo) DeleteMessage(ctx context.Context, id, userID string) error {
	return nil
}

type mockOpenRouterService struct {
	generateStreamFn func(ctx context.Context, messages []openrouter.Message, model *string, temperature float64) (<-chan string, <-chan openrouter.Usage, <-chan error)
}

func (m *mockOpenRouterService) GenerateResponse(ctx context.Context, messages []openrouter.Message, model *string, temperature float64) (*openrouter.Response, error) {
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

	svc := NewChatConversationService(repo, openRouter, &config.Config{
		OpenRouter: config.OpenRouterConfig{DefaultModel: "test-model"},
	})

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
