package model

import (
	"time"
)

// AccountDeletionToken is a one-time token emailed to a user to confirm
// deleting their own account. Token holds the SHA-256 hash, never the raw value.
type AccountDeletionToken struct {
	ID        string     `json:"id" gorm:"type:uuid;primaryKey;default:uuidv7()"`
	UserID    string     `json:"user_id" gorm:"type:uuid;not null;index"`
	Token     string     `json:"token" gorm:"type:text;not null;uniqueIndex"`
	CreatedAt time.Time  `json:"created_at" gorm:"not null;default:now()"`
	ExpiresAt time.Time  `json:"expires_at" gorm:"not null"`
	UsedAt    *time.Time `json:"used_at"`
	User      *User      `json:"-" gorm:"foreignKey:UserID"`
}

func (AccountDeletionToken) TableName() string {
	return "account_deletion_tokens"
}
