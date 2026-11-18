// Package userref is the shared, read-only view of a user that modules embed
// in their own models and responses (authors, owners, members) without
// depending on the full model.User, whose relations reach into every domain.
package userref

import "gorm.io/gorm"

// Ref is a GORM projection of the users table for preloading relations,
// e.g. `Author *userref.Ref gorm:"foreignKey:AuthorID"`. DeletedAt keeps
// GORM's soft-delete scope, so deleted users are never preloaded — the same
// behaviour as preloading model.User.
type Ref struct {
	ID        string         `gorm:"type:uuid;primaryKey"`
	Username  *string        `gorm:"type:varchar(255)"`
	Image     *string
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName maps Ref onto the users table.
func (Ref) TableName() string {
	return "users"
}

// Brief is the public user summary embedded in API responses.
type Brief struct {
	ID       string  `json:"id"`
	Username *string `json:"username"`
	Image    *string `json:"image"`
}

// ToBrief converts a preloaded Ref; nil (or a zero Ref) yields nil.
func ToBrief(u *Ref) *Brief {
	if u == nil || u.ID == "" {
		return nil
	}
	return &Brief{ID: u.ID, Username: u.Username, Image: u.Image}
}

// PreloadBrief limits a user preload to the columns Brief needs.
func PreloadBrief(db *gorm.DB) *gorm.DB {
	return db.Select("id", "username", "image")
}
