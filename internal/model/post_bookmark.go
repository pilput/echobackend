package model

import (
	"time"
)

type PostBookmark struct {
	ID        string     `json:"id" gorm:"type:uuid;primaryKey;default:uuidv7()"`
	PostID    string     `json:"post_id" gorm:"type:uuid;not null;uniqueIndex:idx_post_bookmarks_unique_user_post"`
	UserID    string     `json:"user_id" gorm:"type:uuid;not null;uniqueIndex:idx_post_bookmarks_unique_user_post"`
	FolderID  *string    `json:"folder_id" gorm:"type:uuid;index"`
	Name      *string    `json:"name" gorm:"type:varchar(255)"`
	Notes     *string    `json:"notes"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`

	Post   Post            `gorm:"foreignKey:PostID"`
	User   User            `gorm:"foreignKey:UserID"`
	Folder *BookmarkFolder `gorm:"foreignKey:FolderID"`
}

func (PostBookmark) TableName() string {
	return "post_bookmarks"
}
