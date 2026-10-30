package dto

import (
	"encoding/json"
	"strings"
	"testing"

	"echobackend/internal/model"
)

func TestChatConversationDetailAlwaysHasChatMessagesArray(t *testing.T) {
	conv := &model.ChatConversation{ID: "c-1", UserID: "u-1", Title: "Empty"}

	raw, err := json.Marshal(ChatConversationToDetailResponse(conv))
	if err != nil {
		t.Fatalf("marshal detail: %v", err)
	}
	if !strings.Contains(string(raw), `"chat_messages":[]`) {
		t.Fatalf("detail of an empty conversation must carry chat_messages: [], got %s", raw)
	}

	raw, err = json.Marshal(ChatConversationToResponse(conv))
	if err != nil {
		t.Fatalf("marshal summary: %v", err)
	}
	if strings.Contains(string(raw), "chat_messages") {
		t.Fatalf("list/create/update responses must omit chat_messages, got %s", raw)
	}
}
