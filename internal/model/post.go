package model

import (
	"time"
)

// Post is deliberately not soft-deletable: it has no gorm.DeletedAt field, so
// Delete issues a real DELETE and the child rows go with it through the
// ON DELETE CASCADE foreign keys defined in the migrations.
type Post struct {
	ID            string         `json:"id" gorm:"type:uuid;primaryKey;default:uuidv7()"`
	CreatedAt     *time.Time     `json:"created_at"`
	UpdatedAt     *time.Time     `json:"updated_at"`
	Title         *string        `json:"title" gorm:"type:varchar(255);not null"`
	CreatedBy     *string        `json:"created_by" gorm:"type:uuid;not null;uniqueIndex:creator_and_slug_unique"`
	Body          *string        `json:"body"`
	Slug          *string        `json:"slug" gorm:"type:varchar(255);not null;uniqueIndex:creator_and_slug_unique"`
	PhotoURL      *string        `json:"photo_url"`
	Published     *bool          `json:"published" gorm:"default:true"`
	PublishedAt   *time.Time     `json:"published_at"`
	ViewCount     int64          `json:"view_count" gorm:"type:bigint;default:0"`
	LikeCount     int64          `json:"like_count" gorm:"type:bigint;default:0"`
	BookmarkCount int64          `json:"bookmark_count" gorm:"type:bigint;default:0;check:chk_posts_counts_positive,view_count >= 0 AND like_count >= 0 AND bookmark_count >= 0"`
	PostComments  []PostComment  `gorm:"foreignKey:PostID"`
	PostLikes     []PostLike     `gorm:"foreignKey:PostID"`
	PostBookmarks []PostBookmark `gorm:"foreignKey:PostID"`
	User          *User          `gorm:"foreignKey:CreatedBy;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"user,omitempty"`
	Tags          []Tag          `gorm:"many2many:posts_to_tags;"`
}

func (Post) TableName() string {
	return "posts"
}
