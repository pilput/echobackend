package repository

import (
	"context"
	"time"

	"echobackend/internal/dto"
	"echobackend/internal/model"

	"gorm.io/gorm"
)

type PostViewRepository interface {
	CreateView(ctx context.Context, view *model.PostView) error
	GetViewsByPostID(ctx context.Context, postID string, limit, offset int) ([]*model.PostView, int64, error)
	GetViewStats(ctx context.Context, postID string) (*dto.PostViewStats, error)
	HasUserViewedPost(ctx context.Context, postID, userID string) (bool, error)
	// HasAnonymousViewedSince reports whether an anonymous visitor with this
	// IP + User-Agent viewed the post after since.
	HasAnonymousViewedSince(ctx context.Context, postID, ipAddress, userAgent string, since time.Time) (bool, error)
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

	// Unique views: one viewer per user, or per anonymous IP + User-Agent
	// (the identity RecordView deduplicates anonymous visitors on).
	if err := r.db.WithContext(ctx).Model(&model.PostView{}).
		Select(`COUNT(DISTINCT COALESCE(user_id::text,
			'anon:' || COALESCE(ip_address, '') || '|' || COALESCE(user_agent, '')))`).
		Where("post_id = ?", postID).
		Scan(&stats.UniqueViews).Error; err != nil {
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

func (r *postViewRepository) HasAnonymousViewedSince(ctx context.Context, postID, ipAddress, userAgent string, since time.Time) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.PostView{}).
		Where("post_id = ? AND user_id IS NULL AND ip_address = ? AND user_agent = ? AND created_at > ?",
			postID, ipAddress, userAgent, since).
		Limit(1).
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
