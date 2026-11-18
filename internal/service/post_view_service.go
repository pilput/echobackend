package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/model"
	"echobackend/internal/repository"
	"echobackend/pkg/useragent"
)

type PostViewService interface {
	RecordView(ctx context.Context, postID, userID string, ipAddress, userAgent *string) error
	GetViewsByPostID(ctx context.Context, postID, requesterID string, isAdmin bool, limit, offset int) ([]*dto.PostViewResponse, int64, error)
	GetViewStats(ctx context.Context, postID string) (*dto.PostViewStats, error)
	HasUserViewedPost(ctx context.Context, postID, userID string) (bool, error)
	GetMyPostsAnalytics(ctx context.Context, userID string, q *dto.MyPostsAnalyticsQuery) (*dto.MyPostsAnalyticsResponse, error)
	GetMyPostsLikesByMonth(ctx context.Context, userID string, q *dto.MyPostsLikesByMonthQuery) (*dto.MyPostsLikesByMonthResponse, error)
}

// anonymousDedupWindow is how long an anonymous visitor's view blocks another
// one of the same post.
const anonymousDedupWindow = 24 * time.Hour

// viewedMarkerTTL is how long an "already viewed this post" marker lives in
// the cache. It only short-circuits repeat RecordView calls; the database stays
// authoritative, so an expired or missing marker just falls back to the query.
// It must not outlive anonymousDedupWindow.
const viewedMarkerTTL = anonymousDedupWindow

type postViewCache interface {
	BuildKey(parts ...string) string
	GetJSON(ctx context.Context, key string, dest any) (bool, error)
	SetJSONWithTTL(ctx context.Context, key string, value any, ttl time.Duration) error
}

// maxAnalyticsRange bounds the start_date..end_date span of GetMyPostsAnalytics.
const maxAnalyticsRange = 366 * 24 * time.Hour

type postViewService struct {
	postViewRepo repository.PostViewRepository
	postRepo     repository.PostRepository
	postLikeRepo repository.PostLikeRepository
	cache        postViewCache
}

func NewPostViewService(
	postViewRepo repository.PostViewRepository,
	postRepo repository.PostRepository,
	postLikeRepo repository.PostLikeRepository,
	cache ...postViewCache,
) PostViewService {
	var c postViewCache
	if len(cache) > 0 {
		c = cache[0]
	}
	return &postViewService{
		postViewRepo: postViewRepo,
		postRepo:     postRepo,
		postLikeRepo: postLikeRepo,
		cache:        c,
	}
}

// RecordView records a view of postID. Every reader counts, logged in or
// not; what differs is how repeat views are deduplicated:
//
//	viewer        identity              counted
//	logged in     user id               once per post
//	anonymous     IP + User-Agent       once per post per anonymousDedupWindow
//	unidentified  none (no IP)          every request
//
// A cache marker short-circuits repeat views without a DB lookup; the DB stays
// the source of truth when the marker is missing. Automated clients
// (useragent.IsLikelyBot) are accepted but never stored.
func (s *postViewService) RecordView(ctx context.Context, postID, userID string, ipAddress, userAgent *string) error {
	if postID == "" {
		return apperrors.ErrEmptyPostID
	}
	ip, ua := derefTrimmed(ipAddress), derefTrimmed(userAgent)
	if useragent.IsLikelyBot(ua) {
		return nil
	}

	markerKey := s.viewedMarkerKey(postID, viewerMarkerID(userID, ip, ua))
	if markerKey != "" {
		var viewed bool
		if found, err := s.cache.GetJSON(ctx, markerKey, &viewed); err == nil && found && viewed {
			return nil
		}
	}

	post, err := s.postRepo.GetPostByID(ctx, postID)
	if err != nil {
		return fmt.Errorf("failed to verify post existence: %w", err)
	}
	if err := ensurePostInteractable(post); err != nil {
		return err
	}

	hasViewed, err := s.hasViewed(ctx, postID, userID, ip, ua)
	if err != nil {
		return err
	}
	if hasViewed {
		s.setViewedMarker(ctx, markerKey)
		return nil
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
	if ip != "" {
		view.IPAddress = &ip
	}
	if ua != "" {
		view.UserAgent = &ua
	}

	// view_count is maintained by the trigger_update_view_count_insert
	// trigger on post_views; no app-level increment.
	if err := s.postViewRepo.CreateView(ctx, view); err != nil {
		return fmt.Errorf("failed to create view record: %w", err)
	}
	s.setViewedMarker(ctx, markerKey)
	return nil
}

// hasViewed reports whether the viewer already has a stored view that counts
// against a new one. An anonymous viewer without an IP cannot be identified,
// so every request counts.
func (s *postViewService) hasViewed(ctx context.Context, postID, userID, ip, ua string) (bool, error) {
	switch {
	case userID != "":
		viewed, err := s.postViewRepo.HasUserViewedPost(ctx, postID, userID)
		if err != nil {
			return false, fmt.Errorf("failed to check if user viewed post: %w", err)
		}
		return viewed, nil
	case ip != "":
		viewed, err := s.postViewRepo.HasAnonymousViewedSince(ctx, postID, ip, ua, time.Now().Add(-anonymousDedupWindow))
		if err != nil {
			return false, fmt.Errorf("failed to check if visitor viewed post: %w", err)
		}
		return viewed, nil
	default:
		return false, nil
	}
}

// viewerMarkerID is the viewer identity in the cache marker key: the user ID
// (compatible with the old post_view:<post>:<user> key), or a SHA-256 of
// IP + User-Agent so raw IPs never end up in cache keys. "" when the viewer
// cannot be identified.
func viewerMarkerID(userID, ip, ua string) string {
	if userID != "" {
		return userID
	}
	if ip == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(ip + "|" + ua))
	return "anon:" + hex.EncodeToString(sum[:])
}

func derefTrimmed(v *string) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(*v)
}

// viewedMarkerKey returns "" when there is no cache or no viewer to dedupe on.
func (s *postViewService) viewedMarkerKey(postID, viewerID string) string {
	if s.cache == nil || viewerID == "" {
		return ""
	}
	return s.cache.BuildKey("post_view", postID, viewerID)
}

func (s *postViewService) setViewedMarker(ctx context.Context, key string) {
	if key != "" {
		_ = s.cache.SetJSONWithTTL(ctx, key, true, viewedMarkerTTL)
	}
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
	if err := s.ensurePostViewable(ctx, postID); err != nil {
		return nil, err
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
	if err := s.ensurePostViewable(ctx, postID); err != nil {
		return false, err
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

// ensurePostViewable answers view-stats/viewed lookups for a post that does
// not exist or is hidden with ErrPostNotFound (404) instead of zero stats.
func (s *postViewService) ensurePostViewable(ctx context.Context, postID string) error {
	post, err := s.postRepo.GetPostByID(ctx, postID)
	if err != nil {
		return fmt.Errorf("failed to verify post existence: %w", err)
	}
	return ensurePostInteractable(post)
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
