package model

import (
	"time"

	"gorm.io/gorm"
)

type PostComment struct {
	ID              string         `gorm:"type:uuid;primaryKey;default:uuidv7()" json:"id"`
	CreatedAt       *time.Time     `json:"created_at"`
	UpdatedAt       *time.Time     `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
	Text            string         `json:"text" gorm:"type:text;not null"`
	PostID          string         `json:"post_id" gorm:"type:uuid;not null"`
	ParentCommentID *string        `json:"parent_comment_id" gorm:"type:uuid;index"`
	CreatedBy       string         `json:"created_by" gorm:"type:uuid;not null"`

	User          *User        `gorm:"foreignKey:CreatedBy" json:"user,omitempty"`
	Post          *Post        `gorm:"foreignKey:PostID" json:"post,omitempty"`
	ParentComment *PostComment `gorm:"foreignKey:ParentCommentID" json:"parent_comment,omitempty"`
}

func (PostComment) TableName() string {
	return "post_comments"
}

// Comment is an alias for PostComment to ensure naming consistency across layers.
type Comment = PostComment
