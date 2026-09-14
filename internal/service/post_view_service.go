package service

import (
	"context"
	"fmt"
	"time"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/model"
	"echobackend/internal/repository"
)

type PostViewService interface {
	RecordView(ctx context.Context, postID, userID string, ipAddress, userAgent *string) error
	GetViewsByPostID(ctx context.Context, postID, requesterID string, isAdmin bool, limit, offset int) ([]*dto.PostViewResponse, int64, error)
	GetViewStats(ctx context.Context, postID string) (*dto.PostViewStats, error)
	HasUserViewedPost(ctx context.Context, postID, userID string) (bool, error)
	GetMyPostsAnalytics(ctx context.Context, userID string, q *dto.MyPostsAnalyticsQuery) (*dto.MyPostsAnalyticsResponse, error)
	GetMyPostsLikesByMonth(ctx context.Context, userID string, q *dto.MyPostsLikesByMonthQuery) (*dto.MyPostsLikesByMonthResponse, error)
}

// maxAnalyticsRange bounds the start_date..end_date span of GetMyPostsAnalytics.
const maxAnalyticsRange = 366 * 24 * time.Hour

type postViewService struct {
	postViewRepo repository.PostViewRepository
	postRepo     repository.PostRepository
	postLikeRepo repository.PostLikeRepository
}

func NewPostViewService(
	postViewRepo repository.PostViewRepository,
	postRepo repository.PostRepository,
	postLikeRepo repository.PostLikeRepository,
) PostViewService {
	return &postViewService{
		postViewRepo: postViewRepo,
		postRepo:     postRepo,
		postLikeRepo: postLikeRepo,
	}
}

func (s *postViewService) RecordView(ctx context.Context, postID, userID string, ipAddress, userAgent *string) error {
	if postID == "" {
		return apperrors.ErrEmptyPostID
	}

	if _, err := s.postRepo.GetPostByID(ctx, postID); err != nil {
		return fmt.Errorf("failed to verify post existence: %w", err)
	}

	if userID != "" {
		hasViewed, err := s.postViewRepo.HasUserViewedPost(ctx, postID, userID)
		if err != nil {
			return fmt.Errorf("failed to check if user viewed post: %w", err)
		}
		if hasViewed {
			return nil
		}
	}

	now := time.Now()
	view := &model.PostView{
		PostID:    postID,
		CreatedAt: &now,
		UpdatedAt: &now,
	}

	if userID != "" {
		view.UserID = &userID
	}
	if ipAddress != nil && *ipAddress != "" {
		view.IPAddress = ipAddress
	}
	if userAgent != nil && *userAgent != "" {
		view.UserAgent = userAgent
	}

	if err := s.postViewRepo.CreateView(ctx, view); err != nil {
		return fmt.Errorf("failed to create view record: %w", err)
	}

	// view_count is maintained automatically by the database trigger
	// (trigger_update_view_count_insert on post_views), so no app-level
	// increment is needed here. The previous r.db.Raw("view_count + 1") call
	// was a no-op bug: *gorm.DB does not implement clause.Expression, so the
	// UPDATE errored and RecordView returned a false 500 even though the
	// trigger had already counted the view.
	return nil
}

// GetViewsByPostID lists the raw view records of a post. Only the post's author
// (or a super admin) may see who viewed it.
func (s *postViewService) GetViewsByPostID(ctx context.Context, postID, requesterID string, isAdmin bool, limit, offset int) ([]*dto.PostViewResponse, int64, error) {
	if postID == "" {
		return nil, 0, apperrors.ErrEmptyPostID
	}

	post, err := s.postRepo.GetPostByID(ctx, postID)
	if err != nil {
		return nil, 0, err
	}
	if !isAdmin && (post.CreatedBy == nil || *post.CreatedBy != requesterID) {
		return nil, 0, apperrors.ErrNotAuthor
	}

	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	views, total, err := s.postViewRepo.GetViewsByPostID(ctx, postID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get views by post ID: %w", err)
	}

	responses := make([]*dto.PostViewResponse, len(views))
	for i, view := range views {
		responses[i] = dto.PostViewToResponse(view)
	}

	return responses, total, nil
}

func (s *postViewService) GetViewStats(ctx context.Context, postID string) (*dto.PostViewStats, error) {
	if postID == "" {
		return nil, apperrors.ErrEmptyPostID
	}

	stats, err := s.postViewRepo.GetViewStats(ctx, postID)
	if err != nil {
		return nil, fmt.Errorf("failed to get view stats: %w", err)
	}

	return stats, nil
}

func (s *postViewService) HasUserViewedPost(ctx context.Context, postID, userID string) (bool, error) {
	if postID == "" {
		return false, apperrors.ErrEmptyPostID
	}
	if userID == "" {
		return false, nil
	}

	hasViewed, err := s.postViewRepo.HasUserViewedPost(ctx, postID, userID)
	if err != nil {
		return false, fmt.Errorf("failed to check if user viewed post: %w", err)
	}

	return hasViewed, nil
}

func (s *postViewService) GetMyPostsAnalytics(ctx context.Context, userID string, q *dto.MyPostsAnalyticsQuery) (*dto.MyPostsAnalyticsResponse, error) {
	start := time.Now().AddDate(0, 0, -30)
	end := time.Now()
	if q != nil {
		if q.StartDate != "" {
			if parsed, err := time.Parse("2006-01-02", q.StartDate); err == nil {
				start = parsed
			}
		}
		if q.EndDate != "" {
			if parsed, err := time.Parse("2006-01-02", q.EndDate); err == nil {
				end = parsed
			}
		}
	}
	if start.After(end) {
		start, end = end, start
	}
	// view_trend holds one point per day, so an unbounded range would build
	// an arbitrarily large response.
	if end.Sub(start) > maxAnalyticsRange {
		return nil, apperrors.ErrDateRangeTooLarge
	}

	startKey := start.Format("2006-01-02")
	endKey := end.Format("2006-01-02")

	summary, err := s.postRepo.GetAuthorPostStats(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get author post stats: %w", err)
	}

	topPosts, err := s.postRepo.GetTopPostsByAuthor(ctx, userID, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to get top posts: %w", err)
	}
	if topPosts == nil {
		topPosts = []dto.MyPostPerformance{}
	}

	rows, err := s.postViewRepo.GetViewTrendByAuthor(ctx, userID, startKey, endKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get view trend: %w", err)
	}

	rowMap := make(map[string]int64, len(rows))
	for _, row := range rows {
		rowMap[row.Date] = row.Count
	}

	cumulative, err := s.postViewRepo.CountViewsByAuthorBefore(ctx, userID, startKey)
	if err != nil {
		return nil, fmt.Errorf("failed to count views before period: %w", err)
	}

	viewTrend := make([]dto.MyPostsViewTrendPoint, 0)
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dateKey := d.Format("2006-01-02")
		views := rowMap[dateKey]
		cumulative += views
		viewTrend = append(viewTrend, dto.MyPostsViewTrendPoint{
			Date:            dateKey,
			Views:           views,
			CumulativeViews: cumulative,
		})
	}

	return &dto.MyPostsAnalyticsResponse{
		Summary:   *summary,
		ViewTrend: viewTrend,
		TopPosts:  topPosts,
	}, nil
}

func (s *postViewService) GetMyPostsLikesByMonth(ctx context.Context, userID string, q *dto.MyPostsLikesByMonthQuery) (*dto.MyPostsLikesByMonthResponse, error) {
	months := 12
	if q != nil && q.Months >= 1 && q.Months <= 24 {
		months = q.Months
	}

	now := time.Now()
	loc := now.Location()
	endMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	startMonth := endMonth.AddDate(0, -(months - 1), 0)
	endExclusive := endMonth.AddDate(0, 1, 0)

	rows, err := s.postLikeRepo.GetLikesByMonthByAuthor(ctx, userID, startMonth, endExclusive)
	if err != nil {
		return nil, fmt.Errorf("failed to get likes by month: %w", err)
	}

	rowMap := make(map[string]int64, len(rows))
	for _, row := range rows {
		rowMap[row.Month] = row.Count
	}

	series := make([]dto.MyPostsLikesByMonthPoint, 0, months)
	var total int64
	for i := 0; i < months; i++ {
		monthKey := startMonth.AddDate(0, i, 0).Format("2006-01")
		likes := rowMap[monthKey]
		total += likes
		series = append(series, dto.MyPostsLikesByMonthPoint{
			Month: monthKey,
			Likes: likes,
		})
	}

	return &dto.MyPostsLikesByMonthResponse{
		Months: months,
		Series: series,
		Total:  total,
	}, nil
}
