package model

import "time"

// BookmarkFolder is a user-defined folder for organizing saved posts.
type BookmarkFolder struct {
	ID          string     `json:"id" gorm:"type:uuid;primaryKey;default:uuidv7()"`
	UserID      string     `json:"user_id" gorm:"type:uuid;not null;uniqueIndex:idx_bookmark_folders_user_name"`
	Name        string     `json:"name" gorm:"type:varchar(100);not null;uniqueIndex:idx_bookmark_folders_user_name"`
	Description *string    `json:"description"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`

	User          *User          `gorm:"foreignKey:UserID"`
	PostBookmarks []PostBookmark `gorm:"foreignKey:FolderID"`

	// BookmarkCount, when set, is the folder's total bookmark count loaded by
	// a COUNT query (used where PostBookmarks is not preloaded). Not a column.
	BookmarkCount *int64 `json:"-" gorm:"-"`
}

func (BookmarkFolder) TableName() string {
	return "bookmark_folders"
}
