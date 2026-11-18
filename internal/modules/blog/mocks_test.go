package blog

// Hand-written mocks for the blog tests: each method delegates to a function
// field; unstubbed calls return zero values or panic so unexpected paths are
// obvious.

import (
	"context"
	"time"
)

var (
	_ PostLikeRepository   = (*mockPostLikeRepo)(nil)
	_ PostRepository       = (*mockPostRepo)(nil)
	_ PostViewRepository   = (*mockPostViewRepo)(nil)
	_ TagRepository        = (*mockTagRepo)(nil)
	_ PostReportRepository = (*mockPostReportRepo)(nil)
)

// ---- PostLikeRepository mock --------------------------------------------------

type mockPostLikeRepo struct {
	createLikeFn              func(ctx context.Context, like *PostLike) error
	deleteLikeFn              func(ctx context.Context, postID, userID string) error
	getLikesByPostIDFn        func(ctx context.Context, postID string, limit, offset int) ([]*PostLike, int64, error)
	getLikeStatsFn            func(ctx context.Context, postID string) (*PostLikeStats, error)
	hasUserLikedFn            func(ctx context.Context, postID, userID string) (bool, error)
	getLikeFn                 func(ctx context.Context, postID, userID string) (*PostLike, error)
	getLikesByMonthByAuthorFn func(ctx context.Context, userID string, start, endExclusive time.Time) ([]struct {
		Month string
		Count int64
	}, error)
}

func (m *mockPostLikeRepo) CreateLike(ctx context.Context, like *PostLike) error {
	if m.createLikeFn != nil {
		return m.createLikeFn(ctx, like)
	}
	return nil
}

func (m *mockPostLikeRepo) DeleteLike(ctx context.Context, postID, userID string) error {
	if m.deleteLikeFn != nil {
		return m.deleteLikeFn(ctx, postID, userID)
	}
	return nil
}

func (m *mockPostLikeRepo) GetLikesByPostID(ctx context.Context, postID string, limit, offset int) ([]*PostLike, int64, error) {
	if m.getLikesByPostIDFn != nil {
		return m.getLikesByPostIDFn(ctx, postID, limit, offset)
	}
	return nil, 0, nil
}

func (m *mockPostLikeRepo) GetLikeStats(ctx context.Context, postID string) (*PostLikeStats, error) {
	if m.getLikeStatsFn != nil {
		return m.getLikeStatsFn(ctx, postID)
	}
	return nil, nil
}

func (m *mockPostLikeRepo) HasUserLikedPost(ctx context.Context, postID, userID string) (bool, error) {
	if m.hasUserLikedFn != nil {
		return m.hasUserLikedFn(ctx, postID, userID)
	}
	return false, nil
}

func (m *mockPostLikeRepo) GetLikeByUserAndPost(ctx context.Context, postID, userID string) (*PostLike, error) {
	if m.getLikeFn != nil {
		return m.getLikeFn(ctx, postID, userID)
	}
	return nil, nil
}

func (m *mockPostLikeRepo) GetLikesByMonthByAuthor(ctx context.Context, userID string, start, endExclusive time.Time) ([]struct {
	Month string
	Count int64
}, error) {
	if m.getLikesByMonthByAuthorFn != nil {
		return m.getLikesByMonthByAuthorFn(ctx, userID, start, endExclusive)
	}
	return nil, nil
}

// ---- PostRepository mock ------------------------------------------------------

type mockPostRepo struct {
	getPostByIDFn               func(ctx context.Context, id string) (*Post, error)
	existsFn                    func(ctx context.Context, id string) (bool, error)
	getAuthorPostStatsFn        func(ctx context.Context, userID string) (*MyPostsAnalyticsSummary, error)
	getTopPostsByAuthorFn       func(ctx context.Context, userID string, limit int) ([]MyPostPerformance, error)
	createPostFn                func(ctx context.Context, post *Post) error
	createPostWithTagsFn        func(ctx context.Context, post *Post, tags []Tag) (*Post, error)
	getPostsFn                  func(ctx context.Context, limit int, offset int) ([]*Post, int64, error)
	getPostsFilteredFn          func(ctx context.Context, filter *PostQueryFilter) ([]*Post, int64, error)
	getPostByUsernameFn         func(ctx context.Context, username string, offset int, limit int) ([]*Post, int64, error)
	getPostsRandomFn            func(ctx context.Context, limit int) ([]*Post, error)
	getPostsTrendingFn          func(ctx context.Context, limit int) ([]*Post, error)
	getPostBySlugAndUsernameFn  func(ctx context.Context, slug string, username string) (*Post, error)
	getPostsByCreatedByFn       func(ctx context.Context, createdBy string, offset int, limit int, search string, published, hidden *bool) ([]*Post, int64, error)
	deletePostByIDFn            func(ctx context.Context, id string) error
	deletePostByOwnerFn         func(ctx context.Context, id string, userID string) error
	updatePostWithTagsByOwnerFn func(ctx context.Context, id string, userID string, updates map[string]any, tags []Tag) (*Post, error)
	updatePostFn                func(ctx context.Context, id string, updates map[string]any) (*Post, error)
	updatePostWithTagsFn        func(ctx context.Context, id string, updates map[string]any, tags []Tag) (*Post, error)
	getPostsForSitemapFn        func(ctx context.Context, limit int) ([]*SitemapPost, error)
	searchPostsFn               func(ctx context.Context, keyword string, limit int, offset int) ([]*Post, int64, error)
	getPostsByTagFn             func(ctx context.Context, tag string, limit int, offset int, sortBy string, sortOrder string) ([]*Post, int64, error)
	getPostsForYouFn            func(ctx context.Context, userID string, offset int, limit int) ([]*Post, int64, error)
}

func (m *mockPostRepo) CreatePost(ctx context.Context, post *Post) error {
	if m.createPostFn != nil {
		return m.createPostFn(ctx, post)
	}
	panic("CreatePost not stubbed")
}
func (m *mockPostRepo) CreatePostWithTags(ctx context.Context, post *Post, tags []Tag) (*Post, error) {
	if m.createPostWithTagsFn != nil {
		return m.createPostWithTagsFn(ctx, post, tags)
	}
	panic("CreatePostWithTags not stubbed")
}
func (m *mockPostRepo) GetPosts(ctx context.Context, limit int, offset int) ([]*Post, int64, error) {
	if m.getPostsFn != nil {
		return m.getPostsFn(ctx, limit, offset)
	}
	panic("GetPosts not stubbed")
}
func (m *mockPostRepo) GetPostsFiltered(ctx context.Context, filter *PostQueryFilter) ([]*Post, int64, error) {
	if m.getPostsFilteredFn != nil {
		return m.getPostsFilteredFn(ctx, filter)
	}
	panic("GetPostsFiltered not stubbed")
}
func (m *mockPostRepo) GetPostByUsername(ctx context.Context, username string, offset int, limit int) ([]*Post, int64, error) {
	if m.getPostByUsernameFn != nil {
		return m.getPostByUsernameFn(ctx, username, offset, limit)
	}
	panic("GetPostByUsername not stubbed")
}
func (m *mockPostRepo) GetPostsRandom(ctx context.Context, limit int) ([]*Post, error) {
	if m.getPostsRandomFn != nil {
		return m.getPostsRandomFn(ctx, limit)
	}
	panic("GetPostsRandom not stubbed")
}
func (m *mockPostRepo) GetPostsTrending(ctx context.Context, limit int) ([]*Post, error) {
	if m.getPostsTrendingFn != nil {
		return m.getPostsTrendingFn(ctx, limit)
	}
	panic("GetPostsTrending not stubbed")
}
func (m *mockPostRepo) GetPostByID(ctx context.Context, id string) (*Post, error) {
	if m.getPostByIDFn != nil {
		return m.getPostByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockPostRepo) GetPostBySlugAndUsername(ctx context.Context, slug string, username string) (*Post, error) {
	if m.getPostBySlugAndUsernameFn != nil {
		return m.getPostBySlugAndUsernameFn(ctx, slug, username)
	}
	panic("GetPostBySlugAndUsername not stubbed")
}
func (m *mockPostRepo) GetPostsByCreatedBy(ctx context.Context, createdBy string, offset int, limit int, search string, published, hidden *bool) ([]*Post, int64, error) {
	if m.getPostsByCreatedByFn != nil {
		return m.getPostsByCreatedByFn(ctx, createdBy, offset, limit, search, published, hidden)
	}
	panic("GetPostsByCreatedBy not stubbed")
}
func (m *mockPostRepo) DeletePostByID(ctx context.Context, id string) error {
	if m.deletePostByIDFn != nil {
		return m.deletePostByIDFn(ctx, id)
	}
	panic("DeletePostByID not stubbed")
}
func (m *mockPostRepo) DeletePostByOwner(ctx context.Context, id string, userID string) error {
	if m.deletePostByOwnerFn != nil {
		return m.deletePostByOwnerFn(ctx, id, userID)
	}
	if m.deletePostByIDFn != nil {
		return m.deletePostByIDFn(ctx, id)
	}
	panic("DeletePostByOwner not stubbed")
}
func (m *mockPostRepo) UpdatePostByOwner(ctx context.Context, id string, userID string, updates map[string]any) (*Post, error) {
	return m.UpdatePostWithTagsByOwner(ctx, id, userID, updates, nil)
}
func (m *mockPostRepo) UpdatePostWithTagsByOwner(ctx context.Context, id string, userID string, updates map[string]any, tags []Tag) (*Post, error) {
	if m.updatePostWithTagsByOwnerFn != nil {
		return m.updatePostWithTagsByOwnerFn(ctx, id, userID, updates, tags)
	}
	if m.updatePostWithTagsFn != nil {
		return m.updatePostWithTagsFn(ctx, id, updates, tags)
	}
	if m.updatePostFn != nil {
		return m.updatePostFn(ctx, id, updates)
	}
	panic("UpdatePostWithTagsByOwner not stubbed")
}
func (m *mockPostRepo) UpdatePost(ctx context.Context, id string, updates map[string]any) (*Post, error) {
	if m.updatePostFn != nil {
		return m.updatePostFn(ctx, id, updates)
	}
	panic("UpdatePost not stubbed")
}
func (m *mockPostRepo) UpdatePostWithTags(ctx context.Context, id string, updates map[string]any, tags []Tag) (*Post, error) {
	if m.updatePostWithTagsFn != nil {
		return m.updatePostWithTagsFn(ctx, id, updates, tags)
	}
	if m.updatePostFn != nil {
		return m.updatePostFn(ctx, id, updates)
	}
	panic("UpdatePostWithTags not stubbed")
}
func (m *mockPostRepo) GetPostsForSitemap(ctx context.Context, limit int) ([]*SitemapPost, error) {
	if m.getPostsForSitemapFn != nil {
		return m.getPostsForSitemapFn(ctx, limit)
	}
	panic("GetPostsForSitemap not stubbed")
}
func (m *mockPostRepo) SearchPosts(ctx context.Context, keyword string, limit int, offset int) ([]*Post, int64, error) {
	if m.searchPostsFn != nil {
		return m.searchPostsFn(ctx, keyword, limit, offset)
	}
	panic("SearchPosts not stubbed")
}
func (m *mockPostRepo) GetPostsByTag(ctx context.Context, tag string, limit int, offset int, sortBy string, sortOrder string) ([]*Post, int64, error) {
	if m.getPostsByTagFn != nil {
		return m.getPostsByTagFn(ctx, tag, limit, offset, sortBy, sortOrder)
	}
	panic("GetPostsByTag not stubbed")
}
func (m *mockPostRepo) GetPostsForYou(ctx context.Context, userID string, offset int, limit int) ([]*Post, int64, error) {
	if m.getPostsForYouFn != nil {
		return m.getPostsForYouFn(ctx, userID, offset, limit)
	}
	panic("GetPostsForYou not stubbed")
}
func (m *mockPostRepo) ExistsByID(ctx context.Context, id string) (bool, error) {
	if m.existsFn != nil {
		return m.existsFn(ctx, id)
	}
	return false, nil
}
func (m *mockPostRepo) GetAuthorPostStats(ctx context.Context, userID string) (*MyPostsAnalyticsSummary, error) {
	if m.getAuthorPostStatsFn != nil {
		return m.getAuthorPostStatsFn(ctx, userID)
	}
	return &MyPostsAnalyticsSummary{}, nil
}
func (m *mockPostRepo) GetTopPostsByAuthor(ctx context.Context, userID string, limit int) ([]MyPostPerformance, error) {
	if m.getTopPostsByAuthorFn != nil {
		return m.getTopPostsByAuthorFn(ctx, userID, limit)
	}
	return nil, nil
}

// ---- PostViewRepository mock --------------------------------------------------

type mockPostViewRepo struct {
	getViewTrendByAuthorFn func(ctx context.Context, userID, startDate, endDate string) ([]struct {
		Date  string
		Count int64
	}, error)
	countViewsByAuthorBeforeFn func(ctx context.Context, userID, beforeDate string) (int64, error)
	createViewFn               func(ctx context.Context, view *PostView) error
	hasUserViewedPostFn        func(ctx context.Context, postID, userID string) (bool, error)
	hasAnonymousViewedSinceFn  func(ctx context.Context, postID, ipAddress, userAgent string, since time.Time) (bool, error)
	getViewStatsFn             func(ctx context.Context, postID string) (*PostViewStats, error)
}

func (m *mockPostViewRepo) CreateView(ctx context.Context, view *PostView) error {
	if m.createViewFn != nil {
		return m.createViewFn(ctx, view)
	}
	panic("CreateView not stubbed")
}
func (m *mockPostViewRepo) GetViewsByPostID(ctx context.Context, postID string, limit, offset int) ([]*PostView, int64, error) {
	panic("GetViewsByPostID not stubbed")
}
func (m *mockPostViewRepo) GetViewStats(ctx context.Context, postID string) (*PostViewStats, error) {
	if m.getViewStatsFn != nil {
		return m.getViewStatsFn(ctx, postID)
	}
	panic("GetViewStats not stubbed")
}
func (m *mockPostViewRepo) HasUserViewedPost(ctx context.Context, postID, userID string) (bool, error) {
	if m.hasUserViewedPostFn != nil {
		return m.hasUserViewedPostFn(ctx, postID, userID)
	}
	panic("HasUserViewedPost not stubbed")
}
func (m *mockPostViewRepo) HasAnonymousViewedSince(ctx context.Context, postID, ipAddress, userAgent string, since time.Time) (bool, error) {
	if m.hasAnonymousViewedSinceFn != nil {
		return m.hasAnonymousViewedSinceFn(ctx, postID, ipAddress, userAgent, since)
	}
	panic("HasAnonymousViewedSince not stubbed")
}
func (m *mockPostViewRepo) GetViewByUserAndPost(ctx context.Context, postID, userID string) (*PostView, error) {
	panic("GetViewByUserAndPost not stubbed")
}
func (m *mockPostViewRepo) GetViewTrendByAuthor(ctx context.Context, userID, startDate, endDate string) ([]struct {
	Date  string
	Count int64
}, error) {
	if m.getViewTrendByAuthorFn != nil {
		return m.getViewTrendByAuthorFn(ctx, userID, startDate, endDate)
	}
	return nil, nil
}
func (m *mockPostViewRepo) CountViewsByAuthorBefore(ctx context.Context, userID, beforeDate string) (int64, error) {
	if m.countViewsByAuthorBeforeFn != nil {
		return m.countViewsByAuthorBeforeFn(ctx, userID, beforeDate)
	}
	return 0, nil
}

// ---- TagRepository mock -------------------------------------------------------

type mockTagRepo struct {
	createFn             func(ctx context.Context, tag *Tag) error
	findAllFn            func(ctx context.Context) ([]Tag, error)
	findByIDFn           func(ctx context.Context, id uint) (*Tag, error)
	findByNameFn         func(ctx context.Context, name string) (*Tag, error)
	findOrCreateByNameFn func(ctx context.Context, name string) (*Tag, error)
	getTrendingTags      func(ctx context.Context, limit int) ([]*TrendingTagResponse, error)
	updateFn             func(ctx context.Context, tag *Tag) error
	deleteFn             func(ctx context.Context, id uint) error
}

func (m *mockTagRepo) Create(ctx context.Context, tag *Tag) error {
	if m.createFn != nil {
		return m.createFn(ctx, tag)
	}
	return nil
}
func (m *mockTagRepo) FindAll(ctx context.Context) ([]Tag, error) {
	if m.findAllFn != nil {
		return m.findAllFn(ctx)
	}
	return nil, nil
}
func (m *mockTagRepo) FindByID(ctx context.Context, id uint) (*Tag, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockTagRepo) FindByName(ctx context.Context, name string) (*Tag, error) {
	if m.findByNameFn != nil {
		return m.findByNameFn(ctx, name)
	}
	return nil, nil
}
func (m *mockTagRepo) FindOrCreateByName(ctx context.Context, name string) (*Tag, error) {
	if m.findOrCreateByNameFn != nil {
		return m.findOrCreateByNameFn(ctx, name)
	}
	return nil, nil
}
func (m *mockTagRepo) GetTrendingTags(ctx context.Context, limit int) ([]*TrendingTagResponse, error) {
	if m.getTrendingTags != nil {
		return m.getTrendingTags(ctx, limit)
	}
	panic("GetTrendingTags not stubbed")
}
func (m *mockTagRepo) GetTagsForSitemap(ctx context.Context, limit int) ([]*SitemapTag, error) {
	panic("GetTagsForSitemap not stubbed")
}
func (m *mockTagRepo) Update(ctx context.Context, tag *Tag) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, tag)
	}
	return nil
}
func (m *mockTagRepo) Delete(ctx context.Context, id uint) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

// ---- PostReportRepository mock ------------------------------------------------

type mockPostReportRepo struct {
	createFn            func(ctx context.Context, report *PostReport) error
	listGroupedByPostFn func(ctx context.Context, status string, limit, offset int) ([]PostReportGroup, int64, error)
	listByPostFn        func(ctx context.Context, postID, status string, limit, offset int) ([]*PostReport, int64, error)
	moderateFn          func(ctx context.Context, postID, adminID, action string, note *string) error
	listActionsByPostFn func(ctx context.Context, postID string, limit, offset int) ([]*PostModerationAction, int64, error)
}

func (m *mockPostReportRepo) Create(ctx context.Context, report *PostReport) error {
	if m.createFn != nil {
		return m.createFn(ctx, report)
	}
	panic("Create not stubbed")
}

func (m *mockPostReportRepo) ListGroupedByPost(ctx context.Context, status string, limit, offset int) ([]PostReportGroup, int64, error) {
	if m.listGroupedByPostFn != nil {
		return m.listGroupedByPostFn(ctx, status, limit, offset)
	}
	panic("ListGroupedByPost not stubbed")
}

func (m *mockPostReportRepo) ListByPost(ctx context.Context, postID, status string, limit, offset int) ([]*PostReport, int64, error) {
	if m.listByPostFn != nil {
		return m.listByPostFn(ctx, postID, status, limit, offset)
	}
	panic("ListByPost not stubbed")
}

func (m *mockPostReportRepo) Moderate(ctx context.Context, postID, adminID, action string, note *string) error {
	if m.moderateFn != nil {
		return m.moderateFn(ctx, postID, adminID, action, note)
	}
	panic("Moderate not stubbed")
}

func (m *mockPostReportRepo) ListActionsByPost(ctx context.Context, postID string, limit, offset int) ([]*PostModerationAction, int64, error) {
	if m.listActionsByPostFn != nil {
		return m.listActionsByPostFn(ctx, postID, limit, offset)
	}
	panic("ListActionsByPost not stubbed")
}

func (m *mockPostReportRepo) Stats(ctx context.Context) (*ReportStatsResponse, error) {
	panic("Stats not stubbed")
}
