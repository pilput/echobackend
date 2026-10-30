package dto

import (
	"echobackend/internal/model"
	"time"
)

type CreateChatConversationRequest struct {
	Title string `json:"title" validate:"required,max=255"`
}

type CreateChatConversationStreamRequest struct {
	Title       *string  `json:"title" validate:"omitempty,min=1,max=255"`
	Content     string   `json:"content" validate:"required,min=1,max=10000"`
	Model       *string  `json:"model" validate:"omitempty,max=100,free_model"`
	Temperature *float64 `json:"temperature" validate:"omitempty,gte=0,lte=2"`
}

type UpdateChatConversationRequest struct {
	Title    string `json:"title" validate:"max=255"`
	IsPinned *bool  `json:"is_pinned"`
}

type CreateChatMessageRequest struct {
	Content     string   `json:"content" validate:"required,min=1,max=10000"`
	Role        string   `json:"role" validate:"omitempty,max=20"`
	Model       *string  `json:"model" validate:"omitempty,max=100,free_model"`
	Temperature *float64 `json:"temperature" validate:"omitempty,gte=0,lte=2"`
}

type ChatMessageResponse struct {
	ID               string    `json:"id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	ConversationID   string    `json:"conversation_id"`
	UserID           string    `json:"user_id"`
	Role             string    `json:"role"`
	Content          string    `json:"content"`
	Model            *string   `json:"model,omitempty"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens"`
}

func ChatMessageToResponse(m *model.ChatMessage) *ChatMessageResponse {
	if m == nil {
		return nil
	}
	return &ChatMessageResponse{
		ID:               m.ID,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
		ConversationID:   m.ConversationID,
		UserID:           m.UserID,
		Role:             m.Role,
		Content:          m.Content,
		Model:            m.Model,
		PromptTokens:     m.PromptTokens,
		CompletionTokens: m.CompletionTokens,
		TotalTokens:      m.TotalTokens,
	}
}

type ChatConversationResponse struct {
	ID           string     `json:"id"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	Title        string     `json:"title"`
	IsPinned     bool       `json:"is_pinned"`
	PinnedAt     *time.Time `json:"pinned_at,omitempty"`
	UserID       string     `json:"user_id"`
	MessageCount int        `json:"message_count"`
	// ChatMessages is set only on the detail route, where it is always an
	// array (`[]` for an empty conversation); list/create/update leave it nil
	// so the key is omitted rather than sent as null.
	ChatMessages *[]*ChatMessageResponse `json:"chat_messages,omitempty"`
}

func ChatConversationToResponse(c *model.ChatConversation) *ChatConversationResponse {
	if c == nil {
		return nil
	}
	return &ChatConversationResponse{
		ID:           c.ID,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
		Title:        c.Title,
		IsPinned:     c.IsPinned,
		PinnedAt:     c.PinnedAt,
		UserID:       c.UserID,
		MessageCount: len(c.Messages),
	}
}

func ChatConversationToDetailResponse(c *model.ChatConversation) *ChatConversationResponse {
	resp := ChatConversationToResponse(c)
	if resp == nil {
		return nil
	}
	msgs := chatMessagesToResponses(c.Messages)
	resp.ChatMessages = &msgs
	return resp
}

func chatMessagesToResponses(messages []model.ChatMessage) []*ChatMessageResponse {
	if len(messages) == 0 {
		return []*ChatMessageResponse{}
	}
	responses := make([]*ChatMessageResponse, 0, len(messages))
	for i := range messages {
		responses = append(responses, ChatMessageToResponse(&messages[i]))
	}
	return responses
}

type ChatStreamResult struct {
	UserMessage    *ChatMessageResponse `json:"user_message"`
	ConversationID string               `json:"conversation_id,omitempty"`
}
