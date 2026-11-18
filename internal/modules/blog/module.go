// Package blog is the publishing module: posts, tags, comments, likes,
// views, bookmarks, reports/moderation and the admin post/tag stats.
//
// It is self-contained: models, DTOs, repositories, services, handlers and
// routes all live here. It may depend on core packages (config, core,
// middleware, platform, pkg, apperror) but never on another module or the
// legacy layers. What it needs from core user/notification code comes in
// through the AdminChecker and Notifier interfaces (adapters in internal/di).
package blog

import (
	"time"

	appmiddleware "echobackend/internal/middleware"
	"echobackend/internal/platform/cache"
	"echobackend/internal/platform/storage"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"gorm.io/gorm"
)

// Deps are the shared resources the module needs from the container.
type Deps struct {
	DB *gorm.DB
	// Cache may be nil (Redis disabled); every consumer is nil-safe.
	Cache *cache.RedisCache
	// Storage receives uploaded post images.
	Storage storage.FileUploader
	// Admins answers "is this user a super admin right now" from the DB.
	Admins AdminChecker
	// Notifier delivers comment notifications to post authors.
	Notifier Notifier
}

// Guards are the auth middlewares routes are mounted behind.
type Guards struct {
	// Auth requires a logged-in user.
	Auth echo.MiddlewareFunc
	// OptionalAuth reads the user when a valid token is present.
	OptionalAuth echo.MiddlewareFunc
	// Admin requires a super admin; it must run after Auth.
	Admin echo.MiddlewareFunc
}

// Module holds the wired blog handlers.
type Module struct {
	cache *cache.RedisCache

	post       *PostHandler
	postStats  *PostStatsHandler
	postReport *PostReportHandler
	postView   *PostViewHandler
	postLike   *PostLikeHandler
	comment    *CommentHandler
	tag        *TagHandler
	tagStats   *TagStatsHandler
	bookmark   *BookmarkHandler
}

// New wires the module's repositories, services and handlers.
func New(deps Deps) *Module {
	db := deps.DB

	postRepo := NewPostRepository(db)
	postLikeRepo := NewPostLikeRepository(db)

	tagService := NewTagService(NewTagRepository(db), deps.Cache)
	postService := NewPostService(postRepo, tagService, deps.Storage, deps.Cache)
	postViewService := NewPostViewService(NewPostViewRepository(db), postRepo, postLikeRepo, deps.Cache)
	commentService := NewCommentService(NewCommentRepository(db), postRepo, deps.Notifier)

	return &Module{
		cache:      deps.Cache,
		post:       NewPostHandler(postService, postViewService),
		postStats:  NewPostStatsHandler(NewPostStatsService(NewPostStatsRepository(db))),
		postReport: NewPostReportHandler(NewPostReportService(NewPostReportRepository(db), postRepo)),
		postView:   NewPostViewHandler(postViewService, deps.Admins),
		postLike:   NewPostLikeHandler(NewPostLikeService(postLikeRepo, postRepo)),
		comment:    NewCommentHandler(commentService, deps.Admins),
		tag:        NewTagHandler(tagService),
		tagStats:   NewTagStatsHandler(NewTagStatsService(NewTagStatsRepository(db))),
		bookmark:   NewBookmarkHandler(NewBookmarkService(NewBookmarkRepository(db), postRepo)),
	}
}

// RegisterRoutes mounts the module under api (the /api group).
func (m *Module) RegisterRoutes(api *echo.Group, g Guards) {
	m.registerPostRoutes(api, g)
	m.registerTagRoutes(api, g)
	m.registerBookmarkRoutes(api, g)
}

func (m *Module) registerPostRoutes(api *echo.Group, g Guards) {
	posts := api.Group("/posts")
	reportRateLimit := appmiddleware.FixedWindowRateLimiterWithCache(m.cache, "posts:report", 10, time.Hour)
	// Creating posts is capped at the same 10/hour budget as reporting: cheap
	// enough for legitimate authors, expensive enough to stop spam floods.
	createPostRateLimit := appmiddleware.FixedWindowRateLimiterWithCache(m.cache, "posts:create", 10, time.Hour)
	// Views are open to anonymous visitors, so cap how many a single client
	// can send (dedup already makes repeats no-ops; this limits DB lookups).
	viewRateLimit := appmiddleware.FixedWindowRateLimiterWithCache(m.cache, "posts:view", 60, time.Minute)
	{
		posts.POST("", m.post.CreatePost, g.Auth, createPostRateLimit)
		posts.GET("/stats", m.postStats.GetStats, g.Auth, g.Admin)
		posts.GET("/stats/engagement", m.postStats.GetEngagement, g.Auth, g.Admin)
		posts.GET("/reports", m.postReport.ListReportedPosts, g.Auth, g.Admin)
		posts.GET("/reports/stats", m.postReport.GetReportStats, g.Auth, g.Admin)
		posts.GET("/random", m.post.GetPostsRandom)
		posts.GET("/trending", m.post.GetPostsTrending)
		posts.GET("/me", m.post.GetMyPosts, g.Auth)
		posts.GET("/me/analytics", m.post.GetMyPostsAnalytics, g.Auth)
		posts.GET("/me/analytics/likes-by-month", m.post.GetMyPostsLikesByMonth, g.Auth)
		posts.GET("/me/:id", m.post.GetMyPost, g.Auth)
		posts.PUT("/me/:id", m.post.UpdateMyPost, g.Auth)
		posts.DELETE("/me/:id", m.post.DeleteMyPost, g.Auth)
		posts.GET("/feed/for-you", m.post.GetPostsForYou, g.Auth)
		posts.POST("/image", m.post.UploadImagePosts, g.Auth, middleware.BodyLimit(1*1024*1024))
		posts.GET("/sitemap", m.post.GetPostsForSitemap)
		posts.GET("/username/:username", m.post.GetPostsByUsername)
		posts.GET("/u/:username/:slug", m.post.GetPostBySlugAndUsername)
		posts.GET("/tag/:tag", m.post.GetPostsByTag)
		posts.GET("", m.post.GetPosts)
		posts.PUT("/:id", m.post.UpdatePost, g.Auth, g.Admin)
		posts.DELETE("/:id", m.post.DeletePost, g.Auth, g.Admin)
		posts.GET("/:id", m.post.GetPost, g.Auth, g.Admin)

		// Report and moderation routes
		posts.POST("/:id/reports", m.postReport.ReportPost, g.Auth, reportRateLimit)
		posts.GET("/:id/reports", m.postReport.ListPostReports, g.Auth, g.Admin)
		posts.POST("/:id/moderation", m.postReport.ModeratePost, g.Auth, g.Admin)
		posts.GET("/:id/moderation", m.postReport.ListModerationActions, g.Auth, g.Admin)

		// Comment routes
		posts.GET("/:id/comments", m.comment.GetCommentsByPostID)
		posts.POST("/:id/comments", m.comment.CreateComment, g.Auth)
		posts.PUT("/:id/comments/:comment_id", m.comment.UpdateComment, g.Auth)
		posts.DELETE("/:id/comments/:comment_id", m.comment.DeleteComment, g.Auth)

		// View routes
		posts.POST("/:id/view", m.postView.RecordView, g.OptionalAuth, viewRateLimit)
		posts.GET("/:id/views", m.postView.GetPostViews, g.Auth)
		posts.GET("/:id/view-stats", m.postView.GetPostViewStats)
		posts.GET("/:id/viewed", m.postView.CheckUserViewed, g.Auth)

		// Like routes
		posts.POST("/:id/like", m.postLike.LikePost, g.Auth)
		posts.DELETE("/:id/like", m.postLike.UnlikePost, g.Auth)
		posts.GET("/:id/likes", m.postLike.GetPostLikes)
		posts.GET("/:id/like-stats", m.postLike.GetPostLikeStats)
		posts.GET("/:id/liked", m.postLike.CheckUserLiked, g.Auth)
	}
}

func (m *Module) registerTagRoutes(api *echo.Group, g Guards) {
	tags := api.Group("/tags")
	{
		tags.POST("", m.tag.CreateTag, g.Auth)
		tags.GET("", m.tag.GetTags)
		tags.GET("/stats", m.tagStats.GetStats, g.Auth, g.Admin)
		tags.GET("/trending", m.tag.GetTrendingTags)
		tags.GET("/sitemap", m.tag.GetTagsForSitemap)
		tags.GET("/:id", m.tag.GetTagByID)
		tags.PUT("/:id", m.tag.UpdateTag, g.Auth, g.Admin)
		tags.DELETE("/:id", m.tag.DeleteTag, g.Auth, g.Admin)
	}
}

func (m *Module) registerBookmarkRoutes(api *echo.Group, g Guards) {
	bookmarks := api.Group("/bookmarks", g.Auth)
	{
		bookmarks.POST("/:post_id", m.bookmark.ToggleBookmark)
		bookmarks.GET("", m.bookmark.GetBookmarks)
		bookmarks.PATCH("/:bookmark_id", m.bookmark.UpdateBookmark)
		bookmarks.PATCH("/:bookmark_id/move", m.bookmark.MoveBookmark)
		bookmarks.POST("/folders", m.bookmark.CreateFolder)
		bookmarks.GET("/folders", m.bookmark.GetFolders)
		bookmarks.PATCH("/folders/:folder_id", m.bookmark.UpdateFolder)
		bookmarks.DELETE("/folders/:folder_id", m.bookmark.DeleteFolder)
	}
}
