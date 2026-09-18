package model

import (
	"time"

	"gorm.io/gorm"
)

type ChatConversation struct {
	ID        string         `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	Title     string         `gorm:"type:varchar(255);not null"`
	IsPinned  bool           `json:"is_pinned" gorm:"column:is_pinned;default:false;not null"`
	PinnedAt  *time.Time     `json:"pinned_at" gorm:"column:pinned_at"`
	UserID    string         `gorm:"type:uuid;not null"`
	Messages  []ChatMessage  `gorm:"foreignKey:ConversationID"`
}

func (ChatConversation) TableName() string {
	return "chat_conversations"
}
