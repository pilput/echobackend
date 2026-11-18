package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID             string         `json:"id" gorm:"type:uuid;primaryKey;default:uuidv7()"`
	CreatedAt      *time.Time     `json:"created_at"`
	UpdatedAt      *time.Time     `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
	FirstName      *string        `json:"first_name" gorm:"type:varchar(255)"`
	LastName       *string        `json:"last_name" gorm:"type:varchar(255)"`
	Email          string         `json:"email" gorm:"uniqueIndex;not null;type:varchar(255)"`
	Password       *string        `json:"-" gorm:"type:varchar(255)"`
	Image          *string        `json:"image"`
	IsSuperAdmin   *bool          `json:"-" gorm:"default:false"`
	Username       *string        `json:"username" gorm:"uniqueIndex;type:varchar(255)"`
	GithubID       *int64         `json:"github_id" gorm:"uniqueIndex:users_github_id_unique"`
	FollowersCount int64          `json:"followers_count" gorm:"type:bigint;default:0"`
	FollowingCount int64          `json:"following_count" gorm:"type:bigint;default:0"`
	LastLoggedAt   *time.Time     `json:"last_logged_at"`

	Files     []File       `gorm:"foreignKey:CreatedBy"`
	Profile   *Profile     `gorm:"foreignKey:UserID"`
	Sessions  []Session    `gorm:"foreignKey:UserID"`
	Followers []UserFollow `gorm:"foreignKey:FollowingID"`
	Following []UserFollow `gorm:"foreignKey:FollowerID"`
}

func (User) TableName() string {
	return "users"
}
