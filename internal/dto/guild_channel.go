package dto

import (
	"echobackend/internal/model"
	"time"
	"unicode/utf8"
)

type CreateGuildChannelRequest struct {
	Name     string  `json:"name" validate:"required,min=1,max=100"`
	Topic    *string `json:"topic" validate:"omitempty,max=1024"`
	Position *int    `json:"position" validate:"omitempty,min=0"`
}

type UpdateGuildChannelRequest struct {
	Name     *string `json:"name" validate:"omitempty,min=1,max=100"`
	Topic    *string `json:"topic" validate:"omitempty,max=1024"`
	Position *int    `json:"position" validate:"omitempty,min=0"`
}

type GuildChannelResponse struct {
	ID        string     `json:"id"`
	GuildID   string     `json:"guild_id"`
	Name      string     `json:"name"`
	Topic     *string    `json:"topic"`
	Position  int        `json:"position"`
	CreatedBy *string    `json:"created_by"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

type CreateGuildMessageRequest struct {
	Content   string  `json:"content" validate:"required,max=4000"`
	ReplyToID *string `json:"reply_to_id" validate:"omitempty,uuid"`
}

type UpdateGuildMessageRequest struct {
	Content string `json:"content" validate:"required,max=4000"`
}

type GuildMessageResponse struct {
	ID        string     `json:"id"`
	GuildID   string     `json:"guild_id"`
	ChannelID string     `json:"channel_id"`
	AuthorID  string     `json:"author_id"`
	Author    *UserBrief `json:"author,omitempty"`
	Content   string     `json:"content"`
	ReplyToID *string    `json:"reply_to_id"`
	// ReplyTo is a quote of the parent message; nil once the parent is deleted.
	ReplyTo   *GuildMessageReply `json:"reply_to,omitempty"`
	EditedAt  *time.Time         `json:"edited_at"`
	CreatedAt *time.Time         `json:"created_at"`
}

// GuildMessageReplyPreviewLen caps the quoted parent text: a reply shows a
// one-line preview, not the full (up to 4000-rune) parent.
const GuildMessageReplyPreviewLen = 200

// GuildMessageReply is the parent message quoted above a reply. Content is
// truncated to GuildMessageReplyPreviewLen runes.
type GuildMessageReply struct {
	ID       string     `json:"id"`
	AuthorID string     `json:"author_id"`
	Author   *UserBrief `json:"author,omitempty"`
	Content  string     `json:"content"`
}

// GuildMessageCursorMeta paginates channel history backwards. Pass NextBefore
// as ?before= to load the next (older) page.
type GuildMessageCursorMeta struct {
	Limit      int     `json:"limit"`
	HasMore    bool    `json:"has_more"`
	NextBefore *string `json:"next_before,omitempty"`
}

// GuildEvent is one realtime event pushed to guild subscribers over SSE.
type GuildEvent struct {
	Type      string `json:"type"`
	GuildID   string `json:"guild_id"`
	ChannelID string `json:"channel_id,omitempty"`
	Data      any    `json:"data"`
}

// Guild realtime event types.
const (
	GuildEventChannelCreated = "channel.created"
	GuildEventChannelUpdated = "channel.updated"
	GuildEventChannelDeleted = "channel.deleted"
	GuildEventMessageCreated = "message.created"
	GuildEventMessageUpdated = "message.updated"
	GuildEventMessageDeleted = "message.deleted"
)

func GuildChannelToResponse(ch *model.GuildChannel) *GuildChannelResponse {
	if ch == nil {
		return nil
	}
	return &GuildChannelResponse{
		ID:        ch.ID,
		GuildID:   ch.GuildID,
		Name:      ch.Name,
		Topic:     ch.Topic,
		Position:  ch.Position,
		CreatedBy: ch.CreatedBy,
		CreatedAt: ch.CreatedAt,
		UpdatedAt: ch.UpdatedAt,
	}
}

func GuildMessageToResponse(m *model.GuildChannelMessage, guildID string) *GuildMessageResponse {
	if m == nil {
		return nil
	}
	resp := &GuildMessageResponse{
		ID:        m.ID,
		GuildID:   guildID,
		ChannelID: m.ChannelID,
		AuthorID:  m.AuthorID,
		Author:    UserToBrief(m.Author),
		Content:   m.Content,
		ReplyToID: m.ReplyToID,
		EditedAt:  m.EditedAt,
		CreatedAt: m.CreatedAt,
	}
	if m.ReplyTo != nil && m.ReplyTo.ID != "" {
		resp.ReplyTo = &GuildMessageReply{
			ID:       m.ReplyTo.ID,
			AuthorID: m.ReplyTo.AuthorID,
			Author:   UserToBrief(m.ReplyTo.Author),
			Content:  truncateRunes(m.ReplyTo.Content, GuildMessageReplyPreviewLen),
		}
	}
	return resp
}

func truncateRunes(s string, maxRunes int) string {
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	return string([]rune(s)[:maxRunes])
}
