package repository

import (
	"context"
	"errors"
	"fmt"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TagRepository interface {
	Create(ctx context.Context, tag *model.Tag) error
	FindAll(ctx context.Context) ([]model.Tag, error)
	FindByID(ctx context.Context, id uint) (*model.Tag, error)
	FindByName(ctx context.Context, name string) (*model.Tag, error)
	FindOrCreateByName(ctx context.Context, name string) (*model.Tag, error)
	GetTrendingTags(ctx context.Context, limit int) ([]*dto.TrendingTagResponse, error)
	GetTagsForSitemap(ctx context.Context, limit int) ([]*dto.SitemapTag, error)
	Update(ctx context.Context, tag *model.Tag) error
	Delete(ctx context.Context, id uint) error
}

type tagRepository struct {
	db *gorm.DB
}

func NewTagRepository(db *gorm.DB) TagRepository {
	return &tagRepository{db: db}
}

func (r *tagRepository) Create(ctx context.Context, tag *model.Tag) error {
	if tag == nil {
		return apperrors.ErrTagNameRequired
	}
	result := r.db.WithContext(ctx).Create(tag)
	if result.Error != nil {
		return fmt.Errorf("failed to create tag: %w", result.Error)
	}
	return nil
}

func (r *tagRepository) FindAll(ctx context.Context) ([]model.Tag, error) {
	var tags []model.Tag
	err := r.db.WithContext(ctx).Find(&tags).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find all tags: %w", err)
	}
	return tags, nil
}

func (r *tagRepository) FindByID(ctx context.Context, id uint) (*model.Tag, error) {
	if id == 0 {
		return nil, apperrors.ErrInvalidTagID
	}
	var tag model.Tag
	err := r.db.WithContext(ctx).First(&tag, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrTagNotFound
		}
		return nil, fmt.Errorf("failed to find tag by ID %d: %w", id, err)
	}
	return &tag, nil
}

func (r *tagRepository) FindByName(ctx context.Context, name string) (*model.Tag, error) {
	if name == "" {
		return nil, apperrors.ErrTagNameEmpty
	}
	var tag model.Tag
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&tag).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrTagNotFound
		}
		return nil, fmt.Errorf("failed to find tag by name %s: %w", name, err)
	}
	return &tag, nil
}

// FindOrCreateByName returns the tag named name, inserting it if it does not
// exist yet. Two requests submitting the same new tag concurrently both reach
// the INSERT, so it carries ON CONFLICT DO NOTHING against idx_tags_name: the
// loser writes nothing and gets no RETURNING row (leaving ID zero) instead of a
// unique-violation error, and reads the winner's row back.
func (r *tagRepository) FindOrCreateByName(ctx context.Context, name string) (*model.Tag, error) {
	if name == "" {
		return nil, apperrors.ErrTagNameEmpty
	}

	// Fast path: existing tags are by far the common case, and this keeps the
	// happy path free of write traffic.
	tag, err := r.FindByName(ctx, name)
	if err == nil {
		return tag, nil
	}
	if !errors.Is(err, apperrors.ErrTagNotFound) {
		// A connection or query failure must not be mistaken for "absent" and
		// turned into an INSERT.
		return nil, err
	}

	created := &model.Tag{Name: name}
	if err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "name"}},
			DoNothing: true,
		}).
		Create(created).Error; err != nil {
		return nil, fmt.Errorf("failed to create tag %s: %w", name, err)
	}
	if created.ID != 0 {
		return created, nil
	}

	return r.FindByName(ctx, name)
}

func (r *tagRepository) GetTrendingTags(ctx context.Context, limit int) ([]*dto.TrendingTagResponse, error) {
	var tags []*dto.TrendingTagResponse

	err := r.db.WithContext(ctx).
		Table("tags").
		Select("tags.id, tags.name, COALESCE(SUM(posts.view_count), 0) AS total_views, COALESCE(SUM(posts.like_count), 0) AS total_likes, COALESCE(SUM(posts.like_count * 2 + posts.bookmark_count * 2 + posts.view_count), 0) AS trending_score").
		Joins("INNER JOIN posts_to_tags ON posts_to_tags.tag_id = tags.id").
		Joins("INNER JOIN posts ON posts.id = posts_to_tags.post_id").
		Joins("INNER JOIN users ON users.id = posts.created_by AND users.deleted_at IS NULL").
		Where("posts.published = ?", true).
		Group("tags.id, tags.name").
		Order("trending_score DESC, COUNT(posts_to_tags.post_id) DESC, tags.name ASC").
		Limit(limit).
		Scan(&tags).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get trending tags: %w", err)
	}

	return tags, nil
}

func (r *tagRepository) GetTagsForSitemap(ctx context.Context, limit int) ([]*dto.SitemapTag, error) {
	var sitemapTags []*dto.SitemapTag

	err := r.db.WithContext(ctx).
		Table("tags").
		Select("tags.name, tags.created_at").
		Joins("INNER JOIN posts_to_tags ON posts_to_tags.tag_id = tags.id").
		Joins("INNER JOIN posts ON posts.id = posts_to_tags.post_id").
		// Mirrors GetPostsForSitemap: a tag whose only posts belong to deleted
		// authors must not reach the sitemap either.
		Joins("INNER JOIN users ON users.id = posts.created_by AND users.deleted_at IS NULL").
		Where("posts.published = ?", true).
		Group("tags.id, tags.name, tags.created_at").
		Order("tags.name ASC").
		Limit(limit).
		Find(&sitemapTags).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get tags for sitemap: %w", err)
	}

	return sitemapTags, nil
}

func (r *tagRepository) Update(ctx context.Context, tag *model.Tag) error {
	if tag == nil {
		return apperrors.ErrTagNameRequired
	}
	if tag.ID == 0 {
		return apperrors.ErrInvalidTagID
	}

	result := r.db.WithContext(ctx).Save(tag)
	if result.Error != nil {
		return fmt.Errorf("failed to update tag: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrTagNotFound
	}
	return nil
}

func (r *tagRepository) Delete(ctx context.Context, id uint) error {
	if id == 0 {
		return apperrors.ErrInvalidTagID
	}
	result := r.db.WithContext(ctx).Delete(&model.Tag{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete tag: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrTagNotFound
	}
	return nil
}
