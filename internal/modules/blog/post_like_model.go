package blog

import (
	"echobackend/internal/core/userref"
	"time"
)

type PostLike struct {
	ID        string     `json:"id" gorm:"type:uuid;primaryKey;default:uuidv7()"`
	PostID    string     `json:"post_id" gorm:"type:uuid;not null;uniqueIndex:idx_post_likes_unique_user_post"`
	UserID    string     `json:"user_id" gorm:"type:uuid;not null;uniqueIndex:idx_post_likes_unique_user_post"`
	CreatedAt *time.Time `json:"created_at" gorm:"index"`

	Post *Post        `json:"post" gorm:"foreignKey:PostID"`
	User *userref.Ref `json:"user" gorm:"foreignKey:UserID"`
}

func (PostLike) TableName() string {
	return "post_likes"
}
