package repository

import (
	"context"

	"echobackend/internal/dto"

	"gorm.io/gorm"
)

type TagStatsRepository interface {
	GetTagPerformance(ctx context.Context, limit int) ([]dto.TagPerformance, error)
}

type tagStatsRepository struct {
	db *gorm.DB
}

func NewTagStatsRepository(db *gorm.DB) TagStatsRepository {
	return &tagStatsRepository{db: db}
}

func (r *tagStatsRepository) GetTagPerformance(ctx context.Context, limit int) ([]dto.TagPerformance, error) {
	type tagRow struct {
		ID         int
		Name       string
		PostCount  int64
		TotalViews int64
		TotalLikes int64
	}
	var tagRows []tagRow
	if err := r.db.WithContext(ctx).
		Table("tags").
		Select("tags.id, tags.name, COUNT(posts_to_tags.post_id) AS post_count, COALESCE(SUM(posts.view_count), 0) AS total_views, COALESCE(SUM(posts.like_count), 0) AS total_likes").
		Joins("INNER JOIN posts_to_tags ON tags.id = posts_to_tags.tag_id").
		Joins("INNER JOIN posts ON posts_to_tags.post_id = posts.id").
		Where("posts.published = ?", true).
		Group("tags.id, tags.name").
		Order("COUNT(posts_to_tags.post_id) DESC").
		Limit(limit).
		Scan(&tagRows).Error; err != nil {
		return nil, err
	}

	tagPerformance := make([]dto.TagPerformance, 0, len(tagRows))
	for _, row := range tagRows {
		tagPerformance = append(tagPerformance, dto.TagPerformance(row))
	}
	return tagPerformance, nil
}
