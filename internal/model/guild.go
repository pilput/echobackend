package model

import (
	"time"
)

// Guild is the top-level community entity. It is deliberately not
// soft-deletable: deleting a guild is meant to take its memberships with it
// through the ON DELETE CASCADE foreign keys defined in the migration.
type Guild struct {
	ID          string     `json:"id" gorm:"type:uuid;primaryKey;default:uuidv7()"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
	OwnerID     string     `json:"owner_id" gorm:"type:uuid;not null;index"`
	Name        string     `json:"name" gorm:"type:varchar(100);not null"`
	Slug        string     `json:"slug" gorm:"type:varchar(100);not null;uniqueIndex"`
	Description *string    `json:"description"`
	AvatarURL   *string    `json:"avatar_url"`
	IsPublic    bool       `json:"is_public" gorm:"not null;default:true"`
	// MemberCount is maintained by a database trigger on guild_members.
	// Never write it by hand.
	MemberCount int64 `json:"member_count" gorm:"type:bigint;not null;default:0"`

	Owner   *User         `json:"owner,omitempty" gorm:"foreignKey:OwnerID"`
	Members []GuildMember `json:"-" gorm:"foreignKey:GuildID"`
}

func (Guild) TableName() string {
	return "guilds"
}
