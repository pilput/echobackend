package repository

import (
	"context"
	"echobackend/internal/dto"
	"echobackend/internal/model"

	"gorm.io/gorm"
)

type PostViewRepository interface {
	CreateView(ctx context.Context, view *model.PostView) error
	GetViewsByPostID(ctx context.Context, postID string, limit, offset int) ([]*model.PostView, int64, error)
	GetViewStats(ctx context.Context, postID string) (*dto.PostViewStats, error)
	HasUserViewedPost(ctx context.Context, postID, userID string) (bool, error)
	GetViewByUserAndPost(ctx context.Context, postID, userID string) (*model.PostView, error)
	GetViewTrendByAuthor(ctx context.Context, userID, startDate, endDate string) ([]struct {
		Date  string
		Count int64
	}, error)
	CountViewsByAuthorBefore(ctx context.Context, userID, beforeDate string) (int64, error)
}

type postViewRepository struct {
	db *gorm.DB
}

func NewPostViewRepository(db *gorm.DB) PostViewRepository {
	return &postViewRepository{db: db}
}

func (r *postViewRepository) CreateView(ctx context.Context, view *model.PostView) error {
	return r.db.WithContext(ctx).Create(view).Error
}

func (r *postViewRepository) GetViewsByPostID(ctx context.Context, postID string, limit, offset int) ([]*model.PostView, int64, error) {
	var views []*model.PostView
	var total int64

	// Count total views
	if err := r.db.WithContext(ctx).Model(&model.PostView{}).Where("post_id = ?", postID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated views (user_id only — no user preload needed)
	err := r.db.WithContext(ctx).
		Where("post_id = ?", postID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&views).Error

	return views, total, err
}

func (r *postViewRepository) GetViewStats(ctx context.Context, postID string) (*dto.PostViewStats, error) {
	stats := &dto.PostViewStats{PostID: postID}

	// Total views
	if err := r.db.WithContext(ctx).Model(&model.PostView{}).Where("post_id = ?", postID).Count(&stats.TotalViews).Error; err != nil {
		return nil, err
	}

	// Unique views (distinct user_id where user_id is not null)
	if err := r.db.WithContext(ctx).Model(&model.PostView{}).
		Where("post_id = ? AND user_id IS NOT NULL", postID).
		Distinct("user_id").
		Count(&stats.UniqueViews).Error; err != nil {
		return nil, err
	}

	// Anonymous views
	if err := r.db.WithContext(ctx).Model(&model.PostView{}).
		Where("post_id = ? AND user_id IS NULL", postID).
		Count(&stats.AnonymousViews).Error; err != nil {
		return nil, err
	}

	// Authenticated views
	if err := r.db.WithContext(ctx).Model(&model.PostView{}).
		Where("post_id = ? AND user_id IS NOT NULL", postID).
		Count(&stats.AuthenticatedViews).Error; err != nil {
		return nil, err
	}

	return stats, nil
}

func (r *postViewRepository) HasUserViewedPost(ctx context.Context, postID, userID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.PostView{}).
		Where("post_id = ? AND user_id = ?", postID, userID).
		Count(&count).Error
	return count > 0, err
}

func (r *postViewRepository) GetViewByUserAndPost(ctx context.Context, postID, userID string) (*model.PostView, error) {
	var view model.PostView
	err := r.db.WithContext(ctx).
		Where("post_id = ? AND user_id = ?", postID, userID).
		First(&view).Error
	return &view, err
}

func (r *postViewRepository) GetViewTrendByAuthor(ctx context.Context, userID, startDate, endDate string) ([]struct {
	Date  string
	Count int64
}, error) {
	var rows []struct {
		Date  string
		Count int64
	}
	err := r.db.WithContext(ctx).
		Table("post_views AS pv").
		Select("DATE(pv.created_at) AS date, COUNT(*) AS count").
		Joins("JOIN posts AS p ON p.id = pv.post_id").
		Where("p.created_by = ? AND pv.deleted_at IS NULL", userID).
		Where("DATE(pv.created_at) >= ? AND DATE(pv.created_at) <= ?", startDate, endDate).
		Group("DATE(pv.created_at)").
		Order("DATE(pv.created_at) ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *postViewRepository) CountViewsByAuthorBefore(ctx context.Context, userID, beforeDate string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("post_views AS pv").
		Joins("JOIN posts AS p ON p.id = pv.post_id").
		Where("p.created_by = ? AND pv.deleted_at IS NULL", userID).
		Where("DATE(pv.created_at) < ?", beforeDate).
		Count(&count).Error
	return count, err
}
