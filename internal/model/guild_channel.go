package model

import (
	"time"
)

// GuildDefaultChannelName is the channel every guild is created with.
const GuildDefaultChannelName = "general"

// GuildChannel is a text channel inside a guild. Like guilds it is
// hard-deleted, and deleting it cascades to its messages.
type GuildChannel struct {
	ID        string     `json:"id" gorm:"type:uuid;primaryKey;default:uuidv7()"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
	GuildID   string     `json:"guild_id" gorm:"type:uuid;not null;uniqueIndex:idx_guild_channels_guild_name"`
	Name      string     `json:"name" gorm:"type:varchar(100);not null;uniqueIndex:idx_guild_channels_guild_name"`
	Topic     *string    `json:"topic"`
	Position  int        `json:"position" gorm:"not null;default:0"`
	CreatedBy *string    `json:"created_by" gorm:"type:uuid"`

	Guild *Guild `json:"-" gorm:"foreignKey:GuildID"`
}

func (GuildChannel) TableName() string {
	return "guild_channels"
}

// GuildChannelMessage is a chat message posted to a guild channel. Messages are
// hard-deleted; a reply whose parent is deleted keeps ReplyToID nil. There is
// no UpdatedAt: content is the only mutable column and EditedAt records that.
type GuildChannelMessage struct {
	ID        string     `json:"id" gorm:"type:uuid;primaryKey;default:uuidv7()"`
	CreatedAt *time.Time `json:"created_at"`
	ChannelID string     `json:"channel_id" gorm:"type:uuid;not null"`
	AuthorID  string     `json:"author_id" gorm:"type:uuid;not null"`
	Content   string     `json:"content" gorm:"type:text;not null"`
	ReplyToID *string    `json:"reply_to_id" gorm:"type:uuid"`
	EditedAt  *time.Time `json:"edited_at"`

	Author  *User                `json:"author,omitempty" gorm:"foreignKey:AuthorID"`
	ReplyTo *GuildChannelMessage `json:"reply_to,omitempty" gorm:"foreignKey:ReplyToID"`
}

func (GuildChannelMessage) TableName() string {
	return "guild_channel_messages"
}
