package aichat

import "errors"

// AI chat domain errors.
var (
	ErrChatConversationNotFound = errors.New("chat conversation not found")
	ErrChatMessageNotFound      = errors.New("chat message not found")
	ErrConversationNotOwned     = errors.New("access denied: conversation does not belong to user")
	// ErrChatAIResponseSaveFailed means persisting a streamed AI reply failed;
	// the SSE `error` event shows a generic message, never the underlying DB error.
	ErrChatAIResponseSaveFailed = errors.New("failed to save AI response")
	// ErrAIArticleGenerationUnavailable is returned when AI article generation is not
	// configured (no OpenRouter client, blank OPENROUTER_API_KEY or
	// OPENROUTER_DEFAULT_MODEL); handlers answer 503.
	ErrAIArticleGenerationUnavailable = errors.New("AI article generation is not available")
	// ErrAIEmptyResponse is returned when an AI stream ends without content.
	ErrAIEmptyResponse = errors.New("OpenRouter returned an empty response")
)

// MsgInvalidConversationID / MsgInvalidMessageID are the 400 messages for chat
// path segments (:id, :conversationId, :messageId) that are not UUIDs.
const (
	MsgInvalidConversationID = "Invalid conversation ID"
	MsgInvalidMessageID      = "Invalid message ID"
)
