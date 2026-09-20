package model

import (
	"time"
)

// Guild member roles. The owner row is created together with the guild and is
// the only row that may hold GuildRoleOwner.
const (
	GuildRoleOwner  = "owner"
	GuildRoleAdmin  = "admin"
	GuildRoleMember = "member"
)

// GuildMember joins a user to a guild. Rows are hard-deleted on leave so the
// member_count trigger stays correct.
type GuildMember struct {
	ID        string     `json:"id" gorm:"type:uuid;primaryKey;default:uuidv7()"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
	GuildID   string     `json:"guild_id" gorm:"type:uuid;not null;uniqueIndex:idx_guild_members_unique"`
	UserID    string     `json:"user_id" gorm:"type:uuid;not null;uniqueIndex:idx_guild_members_unique"`
	Role      string     `json:"role" gorm:"type:varchar(20);not null;default:member"`

	Guild *Guild `json:"-" gorm:"foreignKey:GuildID"`
	User  *User  `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

func (GuildMember) TableName() string {
	return "guild_members"
}
