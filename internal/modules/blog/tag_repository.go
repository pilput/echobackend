package blog

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TagRepository interface {
	Create(ctx context.Context, tag *Tag) error
	FindAll(ctx context.Context) ([]Tag, error)
	FindByID(ctx context.Context, id uint) (*Tag, error)
	FindByName(ctx context.Context, name string) (*Tag, error)
	FindOrCreateByName(ctx context.Context, name string) (*Tag, error)
	GetTrendingTags(ctx context.Context, limit int) ([]*TrendingTagResponse, error)
	GetTagsForSitemap(ctx context.Context, limit int) ([]*SitemapTag, error)
	Update(ctx context.Context, tag *Tag) error
	Delete(ctx context.Context, id uint) error
}

type tagRepository struct {
	db *gorm.DB
}

func NewTagRepository(db *gorm.DB) TagRepository {
	return &tagRepository{db: db}
}

func (r *tagRepository) Create(ctx context.Context, tag *Tag) error {
	if tag == nil {
		return ErrTagNameRequired
	}
	result := r.db.WithContext(ctx).Create(tag)
	if result.Error != nil {
		return fmt.Errorf("failed to create tag: %w", result.Error)
	}
	return nil
}

func (r *tagRepository) FindAll(ctx context.Context) ([]Tag, error) {
	var tags []Tag
	err := r.db.WithContext(ctx).Order("name ASC").Find(&tags).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find all tags: %w", err)
	}
	return tags, nil
}

func (r *tagRepository) FindByID(ctx context.Context, id uint) (*Tag, error) {
	if id == 0 {
		return nil, ErrInvalidTagID
	}
	var tag Tag
	err := r.db.WithContext(ctx).First(&tag, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTagNotFound
		}
		return nil, fmt.Errorf("failed to find tag by ID %d: %w", id, err)
	}
	return &tag, nil
}

func (r *tagRepository) FindByName(ctx context.Context, name string) (*Tag, error) {
	if name == "" {
		return nil, ErrTagNameEmpty
	}
	var tag Tag
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&tag).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTagNotFound
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
func (r *tagRepository) FindOrCreateByName(ctx context.Context, name string) (*Tag, error) {
	if name == "" {
		return nil, ErrTagNameEmpty
	}

	// Fast path: existing tags are by far the common case, and this keeps the
	// happy path free of write traffic.
	tag, err := r.FindByName(ctx, name)
	if err == nil {
		return tag, nil
	}
	if !errors.Is(err, ErrTagNotFound) {
		// A connection or query failure must not be mistaken for "absent" and
		// turned into an INSERT.
		return nil, err
	}

	created := &Tag{Name: name}
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

func (r *tagRepository) GetTrendingTags(ctx context.Context, limit int) ([]*TrendingTagResponse, error) {
	var tags []*TrendingTagResponse

	err := r.db.WithContext(ctx).
		Table("tags").
		Select("tags.id, tags.name, COALESCE(SUM(posts.view_count), 0) AS total_views, COALESCE(SUM(posts.like_count), 0) AS total_likes, COALESCE(SUM(posts.like_count * 2 + posts.bookmark_count * 2 + posts.view_count), 0) AS trending_score").
		Joins("INNER JOIN posts_to_tags ON posts_to_tags.tag_id = tags.id").
		Joins("INNER JOIN posts ON posts.id = posts_to_tags.post_id").
		Joins("INNER JOIN users ON users.id = posts.created_by AND users.deleted_at IS NULL").
		Where("posts.published = ? AND posts.hidden_at IS NULL", true).
		Group("tags.id, tags.name").
		Order("trending_score DESC, COUNT(posts_to_tags.post_id) DESC, tags.name ASC").
		Limit(limit).
		Scan(&tags).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get trending tags: %w", err)
	}

	return tags, nil
}

func (r *tagRepository) GetTagsForSitemap(ctx context.Context, limit int) ([]*SitemapTag, error) {
	var sitemapTags []*SitemapTag

	err := r.db.WithContext(ctx).
		Table("tags").
		Select("tags.name, tags.created_at").
		Joins("INNER JOIN posts_to_tags ON posts_to_tags.tag_id = tags.id").
		Joins("INNER JOIN posts ON posts.id = posts_to_tags.post_id").
		// Mirrors GetPostsForSitemap: a tag whose only posts belong to deleted
		// authors must not reach the sitemap either.
		Joins("INNER JOIN users ON users.id = posts.created_by AND users.deleted_at IS NULL").
		Where("posts.published = ? AND posts.hidden_at IS NULL", true).
		Group("tags.id, tags.name, tags.created_at").
		Order("tags.name ASC").
		Limit(limit).
		Find(&sitemapTags).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get tags for sitemap: %w", err)
	}

	return sitemapTags, nil
}

func (r *tagRepository) Update(ctx context.Context, tag *Tag) error {
	if tag == nil {
		return ErrTagNameRequired
	}
	if tag.ID == 0 {
		return ErrInvalidTagID
	}

	// Update only the name of an existing row. Save would write every column
	// (nulling created_at) and, on a missing id, fall back to an upsert that
	// creates a tag with the client-supplied id.
	result := r.db.WithContext(ctx).
		Model(&Tag{}).
		Where("id = ?", tag.ID).
		Update("name", tag.Name)
	if result.Error != nil {
		return fmt.Errorf("failed to update tag: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrTagNotFound
	}
	return nil
}

func (r *tagRepository) Delete(ctx context.Context, id uint) error {
	if id == 0 {
		return ErrInvalidTagID
	}
	result := r.db.WithContext(ctx).Delete(&Tag{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete tag: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrTagNotFound
	}
	return nil
}
