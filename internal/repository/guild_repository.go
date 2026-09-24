package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/model"

	"gorm.io/gorm"
)

type GuildRepository interface {
	// CreateGuild inserts the guild, its owner membership and its #general
	// channel in one transaction.
	CreateGuild(ctx context.Context, guild *model.Guild) error
	FindGuildByID(ctx context.Context, id string) (*model.Guild, error)
	FindGuildBySlug(ctx context.Context, slug string) (*model.Guild, error)
	ListPublicGuilds(ctx context.Context, search string, limit, offset int) ([]*model.Guild, int64, error)
	ListGuildsByMember(ctx context.Context, userID string, limit, offset int) ([]*model.Guild, int64, error)
	UpdateGuild(ctx context.Context, id string, updates map[string]any) error
	DeleteGuild(ctx context.Context, id string) error

	AddMember(ctx context.Context, member *model.GuildMember) error
	FindMember(ctx context.Context, guildID, userID string) (*model.GuildMember, error)
	RemoveMember(ctx context.Context, guildID, userID string) error
	ListMembers(ctx context.Context, guildID string, limit, offset int) ([]*model.GuildMember, int64, error)
}

type guildRepository struct {
	db *gorm.DB
}

func NewGuildRepository(db *gorm.DB) GuildRepository {
	return &guildRepository{db: db}
}

func (r *guildRepository) CreateGuild(ctx context.Context, guild *model.Guild) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(guild).Error; err != nil {
			return err
		}
		// The owner is a member from the start, which is also what makes
		// member_count start at 1 via the guild_members trigger.
		owner := &model.GuildMember{
			GuildID: guild.ID,
			UserID:  guild.OwnerID,
			Role:    model.GuildRoleOwner,
		}
		if err := tx.Create(owner).Error; err != nil {
			return err
		}
		ownerID := guild.OwnerID
		general := &model.GuildChannel{
			GuildID:   guild.ID,
			Name:      model.GuildDefaultChannelName,
			CreatedBy: &ownerID,
		}
		return tx.Create(general).Error
	})
	if err != nil {
		// gorm.Config.TranslateError turns SQLSTATE 23505 into ErrDuplicatedKey;
		// the only unique constraint reachable here is guilds.slug.
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return apperrors.ErrGuildSlugExists
		}
		return fmt.Errorf("failed to create guild: %w", err)
	}
	return nil
}

func (r *guildRepository) FindGuildByID(ctx context.Context, id string) (*model.Guild, error) {
	var guild model.Guild
	err := r.db.WithContext(ctx).
		Preload("Owner", preloadUserBrief).
		Where("id = ?", id).
		First(&guild).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrGuildNotFound
		}
		return nil, fmt.Errorf("failed to find guild: %w", err)
	}
	return &guild, nil
}

func (r *guildRepository) FindGuildBySlug(ctx context.Context, slug string) (*model.Guild, error) {
	var guild model.Guild
	err := r.db.WithContext(ctx).
		Preload("Owner", preloadUserBrief).
		Where("slug = ?", slug).
		First(&guild).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrGuildNotFound
		}
		return nil, fmt.Errorf("failed to find guild by slug: %w", err)
	}
	return &guild, nil
}

func (r *guildRepository) ListPublicGuilds(ctx context.Context, search string, limit, offset int) ([]*model.Guild, int64, error) {
	var guilds []*model.Guild
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Guild{}).Where("is_public = ?", true)
	if search = strings.TrimSpace(search); search != "" {
		// ILIKE with a leading wildcard cannot use the btree index; acceptable
		// while the directory is small, revisit with pg_trgm when it is not.
		// Postgres treats backslash as the escape character by default, which
		// is what escapeLikePattern doubles up.
		pattern := "%" + escapeLikePattern(search) + "%"
		query = query.Where("name ILIKE ? OR slug ILIKE ?", pattern, pattern)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count guilds: %w", err)
	}

	err := query.
		Preload("Owner", preloadUserBrief).
		Order("member_count DESC, created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&guilds).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch guilds: %w", err)
	}
	return guilds, total, nil
}

func (r *guildRepository) ListGuildsByMember(ctx context.Context, userID string, limit, offset int) ([]*model.Guild, int64, error) {
	var guilds []*model.Guild
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Guild{}).
		Joins("JOIN guild_members ON guild_members.guild_id = guilds.id AND guild_members.user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count member guilds: %w", err)
	}

	err := query.
		Preload("Owner", preloadUserBrief).
		Order("guild_members.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&guilds).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch member guilds: %w", err)
	}
	return guilds, total, nil
}

func (r *guildRepository) UpdateGuild(ctx context.Context, id string, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}
	result := r.db.WithContext(ctx).Model(&model.Guild{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update guild: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrGuildNotFound
	}
	return nil
}

func (r *guildRepository) DeleteGuild(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&model.Guild{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete guild: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrGuildNotFound
	}
	return nil
}

func (r *guildRepository) AddMember(ctx context.Context, member *model.GuildMember) error {
	err := r.db.WithContext(ctx).Create(member).Error
	if err != nil {
		// Unique (guild_id, user_id): a concurrent join lost the race.
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return apperrors.ErrAlreadyGuildMember
		}
		return fmt.Errorf("failed to add guild member: %w", err)
	}
	return nil
}

func (r *guildRepository) FindMember(ctx context.Context, guildID, userID string) (*model.GuildMember, error) {
	var member model.GuildMember
	err := r.db.WithContext(ctx).
		Preload("User", preloadUserBrief).
		Where("guild_id = ? AND user_id = ?", guildID, userID).
		First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrNotGuildMember
		}
		return nil, fmt.Errorf("failed to find guild member: %w", err)
	}
	return &member, nil
}

func (r *guildRepository) RemoveMember(ctx context.Context, guildID, userID string) error {
	result := r.db.WithContext(ctx).
		Where("guild_id = ? AND user_id = ?", guildID, userID).
		Delete(&model.GuildMember{})
	if result.Error != nil {
		return fmt.Errorf("failed to remove guild member: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrNotGuildMember
	}
	return nil
}

func (r *guildRepository) ListMembers(ctx context.Context, guildID string, limit, offset int) ([]*model.GuildMember, int64, error) {
	var members []*model.GuildMember
	var total int64

	query := r.db.WithContext(ctx).Model(&model.GuildMember{}).Where("guild_id = ?", guildID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count guild members: %w", err)
	}

	err := query.
		Preload("User", preloadUserBrief).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&members).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch guild members: %w", err)
	}
	return members, total, nil
}

// escapeLikePattern neutralises the LIKE wildcards in user input so that a
// search for "100%" does not match everything. Postgres uses backslash as the
// default LIKE escape character.
func escapeLikePattern(s string) string {
	r := strings.NewReplacer(
		"\\", "\\\\",
		"%", "\\%",
		"_", "\\_",
	)
	return r.Replace(s)
}
