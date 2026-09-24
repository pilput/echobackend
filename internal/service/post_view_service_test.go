package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/model"
)

func TestPostViewService_GetMyPostsAnalytics(t *testing.T) {
	postRepo := &mockPostRepo{
		getAuthorPostStatsFn: func(ctx context.Context, userID string) (*dto.MyPostsAnalyticsSummary, error) {
			return &dto.MyPostsAnalyticsSummary{
				TotalPosts:     3,
				PublishedPosts: 2,
				TotalViews:     100,
				TotalLikes:     15,
			}, nil
		},
		getTopPostsByAuthorFn: func(ctx context.Context, userID string, limit int) ([]dto.MyPostPerformance, error) {
			title := "Top post"
			slug := "top-post"
			return []dto.MyPostPerformance{{
				ID:        validPostID,
				Title:     &title,
				Slug:      &slug,
				ViewCount: 50,
				LikeCount: 10,
			}}, nil
		},
	}
	viewRepo := &mockPostViewRepo{
		countViewsByAuthorBeforeFn: func(ctx context.Context, userID, beforeDate string) (int64, error) {
			return 20, nil
		},
		getViewTrendByAuthorFn: func(ctx context.Context, userID, startDate, endDate string) ([]struct {
			Date  string
			Count int64
		}, error) {
			return []struct {
				Date  string
				Count int64
			}{
				{Date: startDate, Count: 5},
				{Date: endDate, Count: 3},
			}, nil
		},
	}

	svc := NewPostViewService(viewRepo, postRepo, &mockPostLikeRepo{})
	got, err := svc.GetMyPostsAnalytics(context.Background(), validUserID, &dto.MyPostsAnalyticsQuery{
		StartDate: "2026-05-01",
		EndDate:   "2026-05-03",
	})
	if err != nil {
		t.Fatalf("GetMyPostsAnalytics returned error: %v", err)
	}

	if got.Summary.TotalPosts != 3 || got.Summary.TotalViews != 100 {
		t.Fatalf("unexpected summary: %+v", got.Summary)
	}
	if len(got.TopPosts) != 1 || got.TopPosts[0].ViewCount != 50 {
		t.Fatalf("unexpected top posts: %+v", got.TopPosts)
	}
	if len(got.ViewTrend) != 3 {
		t.Fatalf("expected 3 trend points, got %d", len(got.ViewTrend))
	}
	if got.ViewTrend[0].Views != 5 || got.ViewTrend[0].CumulativeViews != 25 {
		t.Fatalf("unexpected first trend point: %+v", got.ViewTrend[0])
	}
	if got.ViewTrend[2].Views != 3 || got.ViewTrend[2].CumulativeViews != 28 {
		t.Fatalf("unexpected last trend point: %+v", got.ViewTrend[2])
	}
}

func TestPostViewService_GetMyPostsAnalytics_RejectsLargeRange(t *testing.T) {
	svc := NewPostViewService(&mockPostViewRepo{}, &mockPostRepo{}, &mockPostLikeRepo{})
	_, err := svc.GetMyPostsAnalytics(context.Background(), validUserID, &dto.MyPostsAnalyticsQuery{
		StartDate: "0001-01-01",
		EndDate:   "2026-05-03",
	})
	if !errors.Is(err, apperrors.ErrDateRangeTooLarge) {
		t.Fatalf("expected ErrDateRangeTooLarge, got %v", err)
	}
}

func TestPostViewService_GetViewsByPostID_RejectsNonAuthor(t *testing.T) {
	postRepo := &mockPostRepo{
		getPostByIDFn: func(ctx context.Context, id string) (*model.Post, error) {
			author := "author-id"
			return &model.Post{ID: id, CreatedBy: &author}, nil
		},
	}

	svc := NewPostViewService(&mockPostViewRepo{}, postRepo, &mockPostLikeRepo{})
	_, _, err := svc.GetViewsByPostID(context.Background(), validPostID, "someone-else", false, 10, 0)
	if !errors.Is(err, apperrors.ErrNotAuthor) {
		t.Fatalf("expected ErrNotAuthor, got %v", err)
	}
}

func TestPostViewService_GetMyPostsLikesByMonth(t *testing.T) {
	likeRepo := &mockPostLikeRepo{
		getLikesByMonthByAuthorFn: func(ctx context.Context, userID string, start, endExclusive time.Time) ([]struct {
			Month string
			Count int64
		}, error) {
			return []struct {
				Month string
				Count int64
			}{
				{Month: start.AddDate(0, 1, 0).Format("2006-01"), Count: 5},
				{Month: start.AddDate(0, 2, 0).Format("2006-01"), Count: 3},
			}, nil
		},
	}

	svc := NewPostViewService(&mockPostViewRepo{}, &mockPostRepo{}, likeRepo)
	got, err := svc.GetMyPostsLikesByMonth(context.Background(), validUserID, &dto.MyPostsLikesByMonthQuery{
		Months: 3,
	})
	if err != nil {
		t.Fatalf("GetMyPostsLikesByMonth returned error: %v", err)
	}

	if got.Months != 3 {
		t.Fatalf("expected months=3, got %d", got.Months)
	}
	if len(got.Series) != 3 {
		t.Fatalf("expected 3 series points, got %d", len(got.Series))
	}
	if got.Total != 8 {
		t.Fatalf("expected total=8, got %d", got.Total)
	}
	if got.Series[1].Likes != 5 || got.Series[2].Likes != 3 {
		t.Fatalf("unexpected series: %+v", got.Series)
	}
}

type fakePostViewCache struct {
	store map[string]any
}

func (f *fakePostViewCache) BuildKey(parts ...string) string {
	return strings.Join(parts, ":")
}

func (f *fakePostViewCache) GetJSON(ctx context.Context, key string, dest any) (bool, error) {
	v, ok := f.store[key]
	if !ok {
		return false, nil
	}
	*dest.(*bool) = v.(bool)
	return true, nil
}

func (f *fakePostViewCache) SetJSONWithTTL(ctx context.Context, key string, value any, ttl time.Duration) error {
	f.store[key] = value
	return nil
}

func TestPostViewService_RecordView_CachesViewedMarker(t *testing.T) {
	ctx := context.Background()
	cache := &fakePostViewCache{store: map[string]any{}}
	creates := 0
	viewRepo := &mockPostViewRepo{
		hasUserViewedPostFn: func(ctx context.Context, postID, userID string) (bool, error) { return false, nil },
		createViewFn: func(ctx context.Context, view *model.PostView) error {
			creates++
			return nil
		},
	}
	svc := NewPostViewService(viewRepo, &mockPostRepo{}, &mockPostLikeRepo{}, cache)

	if err := svc.RecordView(ctx, "post-1", "user-1", nil, nil); err != nil {
		t.Fatalf("first RecordView: %v", err)
	}
	if creates != 1 {
		t.Fatalf("expected 1 view created, got %d", creates)
	}

	// Second view must be answered from the cache without touching the DB.
	viewRepo.hasUserViewedPostFn = nil
	viewRepo.createViewFn = nil
	if err := svc.RecordView(ctx, "post-1", "user-1", nil, nil); err != nil {
		t.Fatalf("second RecordView: %v", err)
	}
}

func TestPostViewService_RecordView_MarksAlreadyViewedFromDB(t *testing.T) {
	ctx := context.Background()
	cache := &fakePostViewCache{store: map[string]any{}}
	viewRepo := &mockPostViewRepo{
		hasUserViewedPostFn: func(ctx context.Context, postID, userID string) (bool, error) { return true, nil },
	}
	svc := NewPostViewService(viewRepo, &mockPostRepo{}, &mockPostLikeRepo{}, cache)

	if err := svc.RecordView(ctx, "post-1", "user-1", nil, nil); err != nil {
		t.Fatalf("RecordView: %v", err)
	}
	if _, ok := cache.store[cache.BuildKey("post_view", "post-1", "user-1")]; !ok {
		t.Fatal("expected viewed marker to be cached after DB hit")
	}
}
