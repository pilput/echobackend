package repository

import (
	"context"
	"echobackend/internal/dto"
	"echobackend/internal/model"
	"time"

	"gorm.io/gorm"
)

type PostLikeRepository interface {
	CreateLike(ctx context.Context, like *model.PostLike) error
	DeleteLike(ctx context.Context, postID, userID string) error
	GetLikesByPostID(ctx context.Context, postID string, limit, offset int) ([]*model.PostLike, int64, error)
	GetLikeStats(ctx context.Context, postID string) (*dto.PostLikeStats, error)
	HasUserLikedPost(ctx context.Context, postID, userID string) (bool, error)
	GetLikeByUserAndPost(ctx context.Context, postID, userID string) (*model.PostLike, error)
	GetLikesByMonthByAuthor(ctx context.Context, userID string, start, endExclusive time.Time) ([]struct {
		Month string
		Count int64
	}, error)
}

type postLikeRepository struct {
	db *gorm.DB
}

func NewPostLikeRepository(db *gorm.DB) PostLikeRepository {
	return &postLikeRepository{db: db}
}

func (r *postLikeRepository) CreateLike(ctx context.Context, like *model.PostLike) error {
	return r.db.WithContext(ctx).Create(like).Error
}

func (r *postLikeRepository) DeleteLike(ctx context.Context, postID, userID string) error {
	return r.db.WithContext(ctx).
		Where("post_id = ? AND user_id = ?", postID, userID).
		Delete(&model.PostLike{}).Error
}

func (r *postLikeRepository) GetLikesByPostID(ctx context.Context, postID string, limit, offset int) ([]*model.PostLike, int64, error) {
	var likes []*model.PostLike
	var total int64

	// Count total likes
	if err := r.db.WithContext(ctx).Model(&model.PostLike{}).Where("post_id = ?", postID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated likes with user information
	err := r.db.WithContext(ctx).
		Preload("User", preloadUserBrief).
		Where("post_id = ?", postID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&likes).Error

	return likes, total, err
}

func (r *postLikeRepository) GetLikeStats(ctx context.Context, postID string) (*dto.PostLikeStats, error) {
	var stats dto.PostLikeStats
	stats.PostID = postID

	// Count total likes
	err := r.db.WithContext(ctx).
		Model(&model.PostLike{}).
		Where("post_id = ?", postID).
		Count(&stats.TotalLikes).Error

	if err != nil {
		return nil, err
	}

	return &stats, nil
}

func (r *postLikeRepository) HasUserLikedPost(ctx context.Context, postID, userID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.PostLike{}).Where("post_id = ? AND user_id = ?", postID, userID).Count(&count).Error
	return count > 0, err
}

func (r *postLikeRepository) GetLikeByUserAndPost(ctx context.Context, postID, userID string) (*model.PostLike, error) {
	var like model.PostLike
	err := r.db.WithContext(ctx).Preload("User", preloadUserBrief).Where("post_id = ? AND user_id = ?", postID, userID).First(&like).Error
	return &like, err
}

func (r *postLikeRepository) GetLikesByMonthByAuthor(ctx context.Context, userID string, start, endExclusive time.Time) ([]struct {
	Month string
	Count int64
}, error) {
	var rows []struct {
		Month string
		Count int64
	}
	err := r.db.WithContext(ctx).
		Table("post_likes AS pl").
		Select("TO_CHAR(DATE_TRUNC('month', pl.created_at), 'YYYY-MM') AS month, COUNT(*) AS count").
		Joins("JOIN posts AS p ON p.id = pl.post_id").
		Where("p.created_by = ?", userID).
		Where("pl.created_at >= ? AND pl.created_at < ?", start, endExclusive).
		Group("DATE_TRUNC('month', pl.created_at)").
		Order("DATE_TRUNC('month', pl.created_at) ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}
