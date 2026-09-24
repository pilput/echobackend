package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/model"

	"gorm.io/gorm"
)

// GuildChannelAccess is everything an access check needs, read in one query.
type GuildChannelAccess struct {
	GuildID  string
	IsPublic bool
	// Role is the caller's guild role, empty when not a member.
	Role string
	// ChannelID is set only when a channel was requested and it belongs to
	// the guild.
	ChannelID *string
}

type GuildChannelRepository interface {
	// ResolveAccess loads the guild by slug together with the caller's role
	// and, when channelID is non-empty, whether that channel is in the guild.
	// userID and channelID may be empty. Returns ErrGuildNotFound for an
	// unknown slug.
	ResolveAccess(ctx context.Context, guildSlug, userID, channelID string) (*GuildChannelAccess, error)
	// IsGuildMember is a cheap existence check for re-validating open streams.
	IsGuildMember(ctx context.Context, guildID, userID string) (bool, error)

	CreateChannel(ctx context.Context, channel *model.GuildChannel) error
	// FindChannel only matches a channel that belongs to guildID, so a channel
	// id from another guild reads as not found.
	FindChannel(ctx context.Context, guildID, channelID string) (*model.GuildChannel, error)
	ListChannels(ctx context.Context, guildID string) ([]*model.GuildChannel, error)
	UpdateChannel(ctx context.Context, guildID, channelID string, updates map[string]any) error
	DeleteChannel(ctx context.Context, guildID, channelID string) error

	CreateMessage(ctx context.Context, message *model.GuildChannelMessage) error
	FindMessage(ctx context.Context, channelID, messageID string) (*model.GuildChannelMessage, error)
	// FindMessageAuthor returns only the author id, for permission checks.
	FindMessageAuthor(ctx context.Context, channelID, messageID string) (string, error)
	MessageExists(ctx context.Context, channelID, messageID string) (bool, error)
	// ListMessages returns up to limit messages newest first; with before set,
	// only messages older than that message id.
	ListMessages(ctx context.Context, channelID, before string, limit int) ([]*model.GuildChannelMessage, error)
	UpdateMessageContent(ctx context.Context, channelID, messageID, content string, editedAt time.Time) error
	DeleteMessage(ctx context.Context, channelID, messageID string) error
}

type guildChannelRepository struct {
	db *gorm.DB
}

func NewGuildChannelRepository(db *gorm.DB) GuildChannelRepository {
	return &guildChannelRepository{db: db}
}

func (r *guildChannelRepository) ResolveAccess(ctx context.Context, guildSlug, userID, channelID string) (*GuildChannelAccess, error) {
	// Empty ids become NULL: comparing a uuid column with '' is a cast error,
	// while "= NULL" simply never matches.
	var row struct {
		GuildID   string
		IsPublic  bool
		Role      string
		ChannelID *string
	}
	result := r.db.WithContext(ctx).Raw(`
		SELECT g.id AS guild_id, g.is_public, COALESCE(gm.role, '') AS role, c.id AS channel_id
		FROM guilds g
		LEFT JOIN guild_members gm ON gm.guild_id = g.id AND gm.user_id = ?
		LEFT JOIN guild_channels c ON c.guild_id = g.id AND c.id = ?
		WHERE g.slug = ?`,
		nullIfEmpty(userID), nullIfEmpty(channelID), guildSlug,
	).Scan(&row)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to resolve guild access: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, apperrors.ErrGuildNotFound
	}
	return &GuildChannelAccess{
		GuildID:   row.GuildID,
		IsPublic:  row.IsPublic,
		Role:      row.Role,
		ChannelID: row.ChannelID,
	}, nil
}

func (r *guildChannelRepository) IsGuildMember(ctx context.Context, guildID, userID string) (bool, error) {
	var exists bool
	err := r.db.WithContext(ctx).
		Raw("SELECT EXISTS (SELECT 1 FROM guild_members WHERE guild_id = ? AND user_id = ?)", guildID, userID).
		Scan(&exists).Error
	if err != nil {
		return false, fmt.Errorf("failed to check guild membership: %w", err)
	}
	return exists, nil
}

func (r *guildChannelRepository) CreateChannel(ctx context.Context, channel *model.GuildChannel) error {
	if err := r.db.WithContext(ctx).Create(channel).Error; err != nil {
		// Unique (guild_id, name).
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return apperrors.ErrGuildChannelNameExists
		}
		return fmt.Errorf("failed to create guild channel: %w", err)
	}
	return nil
}

func (r *guildChannelRepository) FindChannel(ctx context.Context, guildID, channelID string) (*model.GuildChannel, error) {
	var channel model.GuildChannel
	err := r.db.WithContext(ctx).
		Where("id = ? AND guild_id = ?", channelID, guildID).
		First(&channel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrGuildChannelNotFound
		}
		return nil, fmt.Errorf("failed to find guild channel: %w", err)
	}
	return &channel, nil
}

func (r *guildChannelRepository) ListChannels(ctx context.Context, guildID string) ([]*model.GuildChannel, error) {
	var channels []*model.GuildChannel
	err := r.db.WithContext(ctx).
		Where("guild_id = ?", guildID).
		Order("position ASC, created_at ASC").
		Find(&channels).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list guild channels: %w", err)
	}
	return channels, nil
}

func (r *guildChannelRepository) UpdateChannel(ctx context.Context, guildID, channelID string, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}
	result := r.db.WithContext(ctx).
		Model(&model.GuildChannel{}).
		Where("id = ? AND guild_id = ?", channelID, guildID).
		Updates(updates)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return apperrors.ErrGuildChannelNameExists
		}
		return fmt.Errorf("failed to update guild channel: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrGuildChannelNotFound
	}
	return nil
}

func (r *guildChannelRepository) DeleteChannel(ctx context.Context, guildID, channelID string) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND guild_id = ?", channelID, guildID).
		Delete(&model.GuildChannel{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete guild channel: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrGuildChannelNotFound
	}
	return nil
}

func (r *guildChannelRepository) CreateMessage(ctx context.Context, message *model.GuildChannelMessage) error {
	if err := r.db.WithContext(ctx).Omit("Author", "ReplyTo").Create(message).Error; err != nil {
		return fmt.Errorf("failed to create guild message: %w", err)
	}
	return nil
}

func (r *guildChannelRepository) FindMessage(ctx context.Context, channelID, messageID string) (*model.GuildChannelMessage, error) {
	var message model.GuildChannelMessage
	err := r.preloadMessage(r.db.WithContext(ctx)).
		Where("id = ? AND channel_id = ?", messageID, channelID).
		First(&message).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrGuildMessageNotFound
		}
		return nil, fmt.Errorf("failed to find guild message: %w", err)
	}
	return &message, nil
}

func (r *guildChannelRepository) FindMessageAuthor(ctx context.Context, channelID, messageID string) (string, error) {
	var authorIDs []string
	err := r.db.WithContext(ctx).
		Model(&model.GuildChannelMessage{}).
		Where("id = ? AND channel_id = ?", messageID, channelID).
		Limit(1).
		Pluck("author_id", &authorIDs).Error
	if err != nil {
		return "", fmt.Errorf("failed to find guild message author: %w", err)
	}
	if len(authorIDs) == 0 {
		return "", apperrors.ErrGuildMessageNotFound
	}
	return authorIDs[0], nil
}

func (r *guildChannelRepository) MessageExists(ctx context.Context, channelID, messageID string) (bool, error) {
	var exists bool
	err := r.db.WithContext(ctx).
		Raw("SELECT EXISTS (SELECT 1 FROM guild_channel_messages WHERE id = ? AND channel_id = ?)", messageID, channelID).
		Scan(&exists).Error
	if err != nil {
		return false, fmt.Errorf("failed to check guild message: %w", err)
	}
	return exists, nil
}

func (r *guildChannelRepository) ListMessages(ctx context.Context, channelID, before string, limit int) ([]*model.GuildChannelMessage, error) {
	var messages []*model.GuildChannelMessage
	query := r.preloadMessage(r.db.WithContext(ctx)).Where("channel_id = ?", channelID)
	if before != "" {
		// uuidv7 ids sort in insert order, so the id is the cursor.
		query = query.Where("id < ?", before)
	}
	err := query.Order("id DESC").Limit(limit).Find(&messages).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list guild messages: %w", err)
	}
	return messages, nil
}

func (r *guildChannelRepository) UpdateMessageContent(ctx context.Context, channelID, messageID, content string, editedAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&model.GuildChannelMessage{}).
		Where("id = ? AND channel_id = ?", messageID, channelID).
		Updates(map[string]any{"content": content, "edited_at": editedAt})
	if result.Error != nil {
		return fmt.Errorf("failed to update guild message: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrGuildMessageNotFound
	}
	return nil
}

func (r *guildChannelRepository) DeleteMessage(ctx context.Context, channelID, messageID string) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND channel_id = ?", messageID, channelID).
		Delete(&model.GuildChannelMessage{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete guild message: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrGuildMessageNotFound
	}
	return nil
}

// preloadMessage batches the relations: one IN query each, and GORM skips
// the reply queries entirely when no message on the page is a reply.
func (r *guildChannelRepository) preloadMessage(db *gorm.DB) *gorm.DB {
	return db.
		Preload("Author", preloadUserBrief).
		Preload("ReplyTo", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "author_id", "content")
		}).
		Preload("ReplyTo.Author", preloadUserBrief)
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
