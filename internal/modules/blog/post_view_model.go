package blog

import (
	"echobackend/internal/core/userref"
	"time"

	"gorm.io/gorm"
)

type PostView struct {
	ID        string         `json:"id" gorm:"type:uuid;primaryKey;default:uuidv7()"`
	PostID    string         `json:"post_id" gorm:"type:uuid;not null;index"`
	UserID    *string        `json:"user_id" gorm:"type:uuid;index"`
	IPAddress *string        `json:"ip_address" gorm:"type:varchar(45)"`
	UserAgent *string        `json:"user_agent"`
	CreatedAt *time.Time     `json:"created_at" gorm:"index"`
	UpdatedAt *time.Time     `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	Post *Post        `json:"post" gorm:"foreignKey:PostID"`
	User *userref.Ref `json:"-" gorm:"foreignKey:UserID"`
}

func (PostView) TableName() string {
	return "post_views"
}
