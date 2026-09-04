package service

// This file contains shared mock implementations of repository interfaces
// used by the service-layer unit tests. The mocks are deliberately simple:
// each method delegates to a function field on the struct, letting individual
// tests configure only the behavior they care about. Methods that are not
// stubbed return their zero value or panic with a clear message so it is
// obvious when a test reaches an unexpected code path.

import (
	"context"
	"echobackend/internal/dto"
	"echobackend/internal/model"
	"echobackend/internal/repository"
	"time"
)

var (
	_ repository.PostLikeRepository = (*mockPostLikeRepo)(nil)
	_ repository.PostRepository     = (*mockPostRepo)(nil)
	_ repository.PostViewRepository = (*mockPostViewRepo)(nil)
	_ repository.UserRepository     = (*mockUserRepo)(nil)
	_ repository.TagRepository      = (*mockTagRepo)(nil)
	_ repository.HoldingRepository  = (*mockHoldingRepo)(nil)
)

// ---- PostLikeRepository mock --------------------------------------------------

type mockPostLikeRepo struct {
	createLikeFn              func(ctx context.Context, like *model.PostLike) error
	deleteLikeFn              func(ctx context.Context, postID, userID string) error
	getLikesByPostIDFn        func(ctx context.Context, postID string, limit, offset int) ([]*model.PostLike, int64, error)
	getLikeStatsFn            func(ctx context.Context, postID string) (*dto.PostLikeStats, error)
	hasUserLikedFn            func(ctx context.Context, postID, userID string) (bool, error)
	getLikeFn                 func(ctx context.Context, postID, userID string) (*model.PostLike, error)
	getLikesByMonthByAuthorFn func(ctx context.Context, userID string, start, endExclusive time.Time) ([]struct {
		Month string
		Count int64
	}, error)
}

func (m *mockPostLikeRepo) CreateLike(ctx context.Context, like *model.PostLike) error {
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

func (m *mockPostLikeRepo) GetLikesByPostID(ctx context.Context, postID string, limit, offset int) ([]*model.PostLike, int64, error) {
	if m.getLikesByPostIDFn != nil {
		return m.getLikesByPostIDFn(ctx, postID, limit, offset)
	}
	return nil, 0, nil
}

func (m *mockPostLikeRepo) GetLikeStats(ctx context.Context, postID string) (*dto.PostLikeStats, error) {
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

func (m *mockPostLikeRepo) GetLikeByUserAndPost(ctx context.Context, postID, userID string) (*model.PostLike, error) {
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
	getPostByIDFn               func(ctx context.Context, id string) (*model.Post, error)
	existsFn                    func(ctx context.Context, id string) (bool, error)
	getAuthorPostStatsFn        func(ctx context.Context, userID string) (*dto.MyPostsAnalyticsSummary, error)
	getTopPostsByAuthorFn       func(ctx context.Context, userID string, limit int) ([]dto.MyPostPerformance, error)
	createPostFn                func(ctx context.Context, post *model.Post) error
	createPostWithTagsFn        func(ctx context.Context, post *model.Post, tags []model.Tag) (*model.Post, error)
	getPostsFn                  func(ctx context.Context, limit int, offset int) ([]*model.Post, int64, error)
	getPostsFilteredFn          func(ctx context.Context, filter *dto.PostQueryFilter) ([]*model.Post, int64, error)
	getPostByUsernameFn         func(ctx context.Context, username string, offset int, limit int) ([]*model.Post, int64, error)
	getPostsRandomFn            func(ctx context.Context, limit int) ([]*model.Post, error)
	getPostsTrendingFn          func(ctx context.Context, limit int) ([]*model.Post, error)
	getPostBySlugAndUsernameFn  func(ctx context.Context, slug string, username string) (*model.Post, error)
	getPostsByCreatedByFn       func(ctx context.Context, createdBy string, offset int, limit int, search string, published *bool) ([]*model.Post, int64, error)
	deletePostByIDFn            func(ctx context.Context, id string) error
	deletePostByOwnerFn         func(ctx context.Context, id string, userID string) error
	updatePostWithTagsByOwnerFn func(ctx context.Context, id string, userID string, updates map[string]any, tags []model.Tag) (*model.Post, error)
	updatePostFn                func(ctx context.Context, id string, updates map[string]any) (*model.Post, error)
	updatePostWithTagsFn        func(ctx context.Context, id string, updates map[string]any, tags []model.Tag) (*model.Post, error)
	getPostsForSitemapFn        func(ctx context.Context, limit int) ([]*dto.SitemapPost, error)
	searchPostsFn               func(ctx context.Context, keyword string, limit int, offset int) ([]*model.Post, int64, error)
	getPostsByTagFn             func(ctx context.Context, tag string, limit int, offset int) ([]*model.Post, int64, error)
	getPostsForYouFn            func(ctx context.Context, userID string, offset int, limit int) ([]*model.Post, int64, error)
}

func (m *mockPostRepo) CreatePost(ctx context.Context, post *model.Post) error {
	if m.createPostFn != nil {
		return m.createPostFn(ctx, post)
	}
	panic("CreatePost not stubbed")
}
func (m *mockPostRepo) CreatePostWithTags(ctx context.Context, post *model.Post, tags []model.Tag) (*model.Post, error) {
	if m.createPostWithTagsFn != nil {
		return m.createPostWithTagsFn(ctx, post, tags)
	}
	panic("CreatePostWithTags not stubbed")
}
func (m *mockPostRepo) GetPosts(ctx context.Context, limit int, offset int) ([]*model.Post, int64, error) {
	if m.getPostsFn != nil {
		return m.getPostsFn(ctx, limit, offset)
	}
	panic("GetPosts not stubbed")
}
func (m *mockPostRepo) GetPostsFiltered(ctx context.Context, filter *dto.PostQueryFilter) ([]*model.Post, int64, error) {
	if m.getPostsFilteredFn != nil {
		return m.getPostsFilteredFn(ctx, filter)
	}
	panic("GetPostsFiltered not stubbed")
}
func (m *mockPostRepo) GetPostByUsername(ctx context.Context, username string, offset int, limit int) ([]*model.Post, int64, error) {
	if m.getPostByUsernameFn != nil {
		return m.getPostByUsernameFn(ctx, username, offset, limit)
	}
	panic("GetPostByUsername not stubbed")
}
func (m *mockPostRepo) GetPostsRandom(ctx context.Context, limit int) ([]*model.Post, error) {
	if m.getPostsRandomFn != nil {
		return m.getPostsRandomFn(ctx, limit)
	}
	panic("GetPostsRandom not stubbed")
}
func (m *mockPostRepo) GetPostsTrending(ctx context.Context, limit int) ([]*model.Post, error) {
	if m.getPostsTrendingFn != nil {
		return m.getPostsTrendingFn(ctx, limit)
	}
	panic("GetPostsTrending not stubbed")
}
func (m *mockPostRepo) GetPostByID(ctx context.Context, id string) (*model.Post, error) {
	if m.getPostByIDFn != nil {
		return m.getPostByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockPostRepo) GetPostBySlugAndUsername(ctx context.Context, slug string, username string) (*model.Post, error) {
	if m.getPostBySlugAndUsernameFn != nil {
		return m.getPostBySlugAndUsernameFn(ctx, slug, username)
	}
	panic("GetPostBySlugAndUsername not stubbed")
}
func (m *mockPostRepo) GetPostsByCreatedBy(ctx context.Context, createdBy string, offset int, limit int, search string, published *bool) ([]*model.Post, int64, error) {
	if m.getPostsByCreatedByFn != nil {
		return m.getPostsByCreatedByFn(ctx, createdBy, offset, limit, search, published)
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
func (m *mockPostRepo) UpdatePostByOwner(ctx context.Context, id string, userID string, updates map[string]any) (*model.Post, error) {
	return m.UpdatePostWithTagsByOwner(ctx, id, userID, updates, nil)
}
func (m *mockPostRepo) UpdatePostWithTagsByOwner(ctx context.Context, id string, userID string, updates map[string]any, tags []model.Tag) (*model.Post, error) {
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
func (m *mockPostRepo) UpdatePost(ctx context.Context, id string, updates map[string]any) (*model.Post, error) {
	if m.updatePostFn != nil {
		return m.updatePostFn(ctx, id, updates)
	}
	panic("UpdatePost not stubbed")
}
func (m *mockPostRepo) UpdatePostWithTags(ctx context.Context, id string, updates map[string]any, tags []model.Tag) (*model.Post, error) {
	if m.updatePostWithTagsFn != nil {
		return m.updatePostWithTagsFn(ctx, id, updates, tags)
	}
	if m.updatePostFn != nil {
		return m.updatePostFn(ctx, id, updates)
	}
	panic("UpdatePostWithTags not stubbed")
}
func (m *mockPostRepo) GetPostsForSitemap(ctx context.Context, limit int) ([]*dto.SitemapPost, error) {
	if m.getPostsForSitemapFn != nil {
		return m.getPostsForSitemapFn(ctx, limit)
	}
	panic("GetPostsForSitemap not stubbed")
}
func (m *mockPostRepo) SearchPosts(ctx context.Context, keyword string, limit int, offset int) ([]*model.Post, int64, error) {
	if m.searchPostsFn != nil {
		return m.searchPostsFn(ctx, keyword, limit, offset)
	}
	panic("SearchPosts not stubbed")
}
func (m *mockPostRepo) GetPostsByTag(ctx context.Context, tag string, limit int, offset int) ([]*model.Post, int64, error) {
	if m.getPostsByTagFn != nil {
		return m.getPostsByTagFn(ctx, tag, limit, offset)
	}
	panic("GetPostsByTag not stubbed")
}
func (m *mockPostRepo) GetPostsForYou(ctx context.Context, userID string, offset int, limit int) ([]*model.Post, int64, error) {
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
func (m *mockPostRepo) GetAuthorPostStats(ctx context.Context, userID string) (*dto.MyPostsAnalyticsSummary, error) {
	if m.getAuthorPostStatsFn != nil {
		return m.getAuthorPostStatsFn(ctx, userID)
	}
	return &dto.MyPostsAnalyticsSummary{}, nil
}
func (m *mockPostRepo) GetTopPostsByAuthor(ctx context.Context, userID string, limit int) ([]dto.MyPostPerformance, error) {
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
}

func (m *mockPostViewRepo) CreateView(ctx context.Context, view *model.PostView) error {
	panic("CreateView not stubbed")
}
func (m *mockPostViewRepo) GetViewsByPostID(ctx context.Context, postID string, limit, offset int) ([]*model.PostView, int64, error) {
	panic("GetViewsByPostID not stubbed")
}
func (m *mockPostViewRepo) GetViewStats(ctx context.Context, postID string) (*dto.PostViewStats, error) {
	panic("GetViewStats not stubbed")
}
func (m *mockPostViewRepo) HasUserViewedPost(ctx context.Context, postID, userID string) (bool, error) {
	panic("HasUserViewedPost not stubbed")
}
func (m *mockPostViewRepo) GetViewByUserAndPost(ctx context.Context, postID, userID string) (*model.PostView, error) {
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

// ---- UserRepository mock ------------------------------------------------------

type mockUserRepo struct {
	getByIDFn             func(ctx context.Context, id string, deletedOnly bool) (*model.User, error)
	getByUsernameFn       func(ctx context.Context, username string) (*model.User, error)
	getUsersFn            func(ctx context.Context, offset int, limit int, deletedFilter dto.UserDeletedFilter) ([]*model.User, int64, error)
	softDeleteFn          func(ctx context.Context, id string) error
	restoreByIDFn         func(ctx context.Context, id string) error
	createFn              func(ctx context.Context, user *model.User) error
	updateFn              func(ctx context.Context, user *model.User) error
	existsFn              func(ctx context.Context, email string) (bool, error)
	getByEmailFn          func(ctx context.Context, email string) (*model.User, error)
	checkUserByUsernameFn func(ctx context.Context, username string) error
}

func (m *mockUserRepo) Create(ctx context.Context, user *model.User) error {
	if m.createFn != nil {
		return m.createFn(ctx, user)
	}
	return nil
}
func (m *mockUserRepo) GetByID(ctx context.Context, id string, deletedOnly bool) (*model.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id, deletedOnly)
	}
	return nil, nil
}
func (m *mockUserRepo) GetUsers(ctx context.Context, offset int, limit int, deletedFilter dto.UserDeletedFilter) ([]*model.User, int64, error) {
	if m.getUsersFn != nil {
		return m.getUsersFn(ctx, offset, limit, deletedFilter)
	}
	return nil, 0, nil
}
func (m *mockUserRepo) GetUsersByEmail(ctx context.Context, email string) ([]*model.User, error) {
	panic("GetUsersByEmail not stubbed")
}
func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return nil, nil
}
func (m *mockUserRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	if m.getByUsernameFn != nil {
		return m.getByUsernameFn(ctx, username)
	}
	return nil, nil
}
func (m *mockUserRepo) Update(ctx context.Context, user *model.User) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, user)
	}
	return nil
}
func (m *mockUserRepo) SoftDeleteByID(ctx context.Context, id string) error {
	if m.softDeleteFn != nil {
		return m.softDeleteFn(ctx, id)
	}
	return nil
}
func (m *mockUserRepo) RestoreByID(ctx context.Context, id string) error {
	if m.restoreByIDFn != nil {
		return m.restoreByIDFn(ctx, id)
	}
	return nil
}
func (m *mockUserRepo) Exists(ctx context.Context, email string) (bool, error) {
	if m.existsFn != nil {
		return m.existsFn(ctx, email)
	}
	return false, nil
}
func (m *mockUserRepo) CheckUserByUsername(ctx context.Context, username string) error {
	if m.checkUserByUsernameFn != nil {
		return m.checkUserByUsernameFn(ctx, username)
	}
	return nil
}

// ---- TagRepository mock -------------------------------------------------------

type mockTagRepo struct {
	createFn        func(ctx context.Context, tag *model.Tag) error
	findAllFn       func(ctx context.Context) ([]model.Tag, error)
	findByIDFn      func(ctx context.Context, id uint) (*model.Tag, error)
	findByNameFn    func(ctx context.Context, name string) (*model.Tag, error)
	getTrendingTags func(ctx context.Context, limit int) ([]*dto.TrendingTagResponse, error)
	updateFn        func(ctx context.Context, tag *model.Tag) error
	deleteFn        func(ctx context.Context, id uint) error
}

func (m *mockTagRepo) Create(ctx context.Context, tag *model.Tag) error {
	if m.createFn != nil {
		return m.createFn(ctx, tag)
	}
	return nil
}
func (m *mockTagRepo) FindAll(ctx context.Context) ([]model.Tag, error) {
	if m.findAllFn != nil {
		return m.findAllFn(ctx)
	}
	return nil, nil
}
func (m *mockTagRepo) FindByID(ctx context.Context, id uint) (*model.Tag, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockTagRepo) FindByName(ctx context.Context, name string) (*model.Tag, error) {
	if m.findByNameFn != nil {
		return m.findByNameFn(ctx, name)
	}
	return nil, nil
}
func (m *mockTagRepo) GetTrendingTags(ctx context.Context, limit int) ([]*dto.TrendingTagResponse, error) {
	if m.getTrendingTags != nil {
		return m.getTrendingTags(ctx, limit)
	}
	panic("GetTrendingTags not stubbed")
}
func (m *mockTagRepo) GetTagsForSitemap(ctx context.Context, limit int) ([]*dto.SitemapTag, error) {
	panic("GetTagsForSitemap not stubbed")
}
func (m *mockTagRepo) Update(ctx context.Context, tag *model.Tag) error {
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

// ---- HoldingRepository mock ---------------------------------------------------

type holdingFilter = struct {
	Month     *int
	Year      *int
	SortBy    string
	SortOrder string
}

type breakdownRow = struct {
	Name         string
	Invested     float64
	CurrentValue float64
}

type trendRow = struct {
	Month        int
	Year         int
	Invested     float64
	CurrentValue float64
}

type monthlyRow = struct {
	Month        int
	Year         int
	Invested     float64
	CurrentValue float64
	Count        int64
}

type mockHoldingRepo struct {
	findAllFn              func(ctx context.Context, userID string, filter *holdingFilter) ([]model.Holding, error)
	findByIDFn             func(ctx context.Context, id int64, userID string) (*model.Holding, error)
	createFn               func(ctx context.Context, h *model.Holding) error
	updateFn               func(ctx context.Context, h *model.Holding) error
	deleteFn               func(ctx context.Context, id int64, userID string) error
	findHoldingTypesFn     func(ctx context.Context) ([]model.HoldingType, error)
	findHoldingTypeByIDFn  func(ctx context.Context, id int) (*model.HoldingType, error)
	findForSyncFn          func(ctx context.Context, userID string, month, year int) ([]model.Holding, error)
	updateFieldsFn         func(ctx context.Context, id int64, userID string, fields map[string]any) error
	findForDuplicateFn     func(ctx context.Context, userID string, month, year int) ([]model.Holding, error)
	deleteByMonthYearFn    func(ctx context.Context, userID string, month, year int) error
	countByMonthYearFn     func(ctx context.Context, userID string, month, year int) (int64, error)
	getSummaryFn           func(ctx context.Context, userID string, month, year *int) (float64, float64, int64, error)
	getTypeBreakdownFn     func(ctx context.Context, userID string, month, year *int) ([]breakdownRow, error)
	getPlatformBreakdownFn func(ctx context.Context, userID string, month, year *int) ([]breakdownRow, error)
	getTrendsFn            func(ctx context.Context, userID string, years []int) ([]trendRow, error)
	getMonthlyDataFn       func(ctx context.Context, userID string, sm, sy, em, ey int) ([]monthlyRow, error)
	findStockSymbolsFn     func(ctx context.Context, userID string) ([]string, error)
}

func (m *mockHoldingRepo) FindAll(ctx context.Context, userID string, filter *holdingFilter) ([]model.Holding, error) {
	if m.findAllFn != nil {
		return m.findAllFn(ctx, userID, filter)
	}
	return nil, nil
}
func (m *mockHoldingRepo) FindByID(ctx context.Context, id int64, userID string) (*model.Holding, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id, userID)
	}
	return nil, nil
}
func (m *mockHoldingRepo) Create(ctx context.Context, h *model.Holding) error {
	if m.createFn != nil {
		return m.createFn(ctx, h)
	}
	return nil
}
func (m *mockHoldingRepo) Update(ctx context.Context, h *model.Holding) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, h)
	}
	return nil
}
func (m *mockHoldingRepo) Delete(ctx context.Context, id int64, userID string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id, userID)
	}
	return nil
}
func (m *mockHoldingRepo) FindHoldingTypes(ctx context.Context) ([]model.HoldingType, error) {
	if m.findHoldingTypesFn != nil {
		return m.findHoldingTypesFn(ctx)
	}
	return nil, nil
}
func (m *mockHoldingRepo) FindHoldingTypeByID(ctx context.Context, id int) (*model.HoldingType, error) {
	if m.findHoldingTypeByIDFn != nil {
		return m.findHoldingTypeByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockHoldingRepo) FindForSync(ctx context.Context, userID string, month, year int) ([]model.Holding, error) {
	if m.findForSyncFn != nil {
		return m.findForSyncFn(ctx, userID, month, year)
	}
	return nil, nil
}
func (m *mockHoldingRepo) UpdateFields(ctx context.Context, id int64, userID string, fields map[string]any) error {
	if m.updateFieldsFn != nil {
		return m.updateFieldsFn(ctx, id, userID, fields)
	}
	return nil
}
func (m *mockHoldingRepo) FindForDuplicate(ctx context.Context, userID string, month, year int) ([]model.Holding, error) {
	if m.findForDuplicateFn != nil {
		return m.findForDuplicateFn(ctx, userID, month, year)
	}
	return nil, nil
}
func (m *mockHoldingRepo) DeleteByUserMonthYear(ctx context.Context, userID string, month, year int) error {
	if m.deleteByMonthYearFn != nil {
		return m.deleteByMonthYearFn(ctx, userID, month, year)
	}
	return nil
}
func (m *mockHoldingRepo) CountByUserMonthYear(ctx context.Context, userID string, month, year int) (int64, error) {
	if m.countByMonthYearFn != nil {
		return m.countByMonthYearFn(ctx, userID, month, year)
	}
	return 0, nil
}
func (m *mockHoldingRepo) GetSummary(ctx context.Context, userID string, month, year *int) (float64, float64, int64, error) {
	if m.getSummaryFn != nil {
		return m.getSummaryFn(ctx, userID, month, year)
	}
	return 0, 0, 0, nil
}
func (m *mockHoldingRepo) GetTypeBreakdown(ctx context.Context, userID string, month, year *int) ([]breakdownRow, error) {
	if m.getTypeBreakdownFn != nil {
		return m.getTypeBreakdownFn(ctx, userID, month, year)
	}
	return nil, nil
}
func (m *mockHoldingRepo) GetPlatformBreakdown(ctx context.Context, userID string, month, year *int) ([]breakdownRow, error) {
	if m.getPlatformBreakdownFn != nil {
		return m.getPlatformBreakdownFn(ctx, userID, month, year)
	}
	return nil, nil
}
func (m *mockHoldingRepo) GetTrends(ctx context.Context, userID string, years []int) ([]trendRow, error) {
	if m.getTrendsFn != nil {
		return m.getTrendsFn(ctx, userID, years)
	}
	return nil, nil
}
func (m *mockHoldingRepo) GetMonthlyData(ctx context.Context, userID string, sm, sy, em, ey int) ([]monthlyRow, error) {
	if m.getMonthlyDataFn != nil {
		return m.getMonthlyDataFn(ctx, userID, sm, sy, em, ey)
	}
	return nil, nil
}
func (m *mockHoldingRepo) FindStockSymbols(ctx context.Context, userID string) ([]string, error) {
	if m.findStockSymbolsFn != nil {
		return m.findStockSymbolsFn(ctx, userID)
	}
	return nil, nil
}
