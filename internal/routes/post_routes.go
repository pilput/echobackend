package routes

import (
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func (r *Routes) setupPostRoutes(api *echo.Group) {
	posts := api.Group("/posts")
	{
		posts.POST("", r.postHandler.CreatePost, r.authMiddleware.Auth())
		posts.GET("/random", r.postHandler.GetPostsRandom)
		posts.GET("/trending", r.postHandler.GetPostsTrending)
		posts.GET("/me", r.postHandler.GetMyPosts, r.authMiddleware.Auth())
		posts.GET("/me/analytics", r.postHandler.GetMyPostsAnalytics, r.authMiddleware.Auth())
		posts.GET("/me/analytics/likes-by-month", r.postHandler.GetMyPostsLikesByMonth, r.authMiddleware.Auth())
		posts.GET("/me/:id", r.postHandler.GetMyPost, r.authMiddleware.Auth())
		posts.PUT("/me/:id", r.postHandler.UpdateMyPost, r.authMiddleware.Auth())
		posts.DELETE("/me/:id", r.postHandler.DeleteMyPost, r.authMiddleware.Auth())
		posts.GET("/feed/for-you", r.postHandler.GetPostsForYou, r.authMiddleware.Auth())
		posts.POST("/image", r.postHandler.UploadImagePosts, r.authMiddleware.Auth(), middleware.BodyLimit(1*1024*1024))
		posts.GET("/sitemap", r.postHandler.GetPostsForSitemap)
		posts.GET("/username/:username", r.postHandler.GetPostsByUsername)
		posts.GET("/u/:username/:slug", r.postHandler.GetPostBySlugAndUsername)
		posts.GET("/tag/:tag", r.postHandler.GetPostsByTag)
		posts.GET("", r.postHandler.GetPosts)
		posts.PUT("/:id", r.postHandler.UpdatePost, r.authMiddleware.Auth(), r.authMiddleware.AuthAdmin())
		posts.DELETE("/:id", r.postHandler.DeletePost, r.authMiddleware.Auth(), r.authMiddleware.AuthAdmin())
		posts.GET("/:id", r.postHandler.GetPost, r.authMiddleware.Auth(), r.authMiddleware.AuthAdmin())

		// Comment routes
		posts.GET("/:id/comments", r.commentHandler.GetCommentsByPostID)
		posts.POST("/:id/comments", r.commentHandler.CreateComment, r.authMiddleware.Auth())
		posts.PUT("/:id/comments/:comment_id", r.commentHandler.UpdateComment, r.authMiddleware.Auth())
		posts.DELETE("/:id/comments/:comment_id", r.commentHandler.DeleteComment, r.authMiddleware.Auth())

		// View routes
		posts.POST("/:id/view", r.postViewHandler.RecordView, r.authMiddleware.Auth()) // Only authenticated users
		posts.GET("/:id/views", r.postViewHandler.GetPostViews, r.authMiddleware.Auth())
		posts.GET("/:id/view-stats", r.postViewHandler.GetPostViewStats)
		posts.GET("/:id/viewed", r.postViewHandler.CheckUserViewed, r.authMiddleware.Auth())

		// Like routes
		posts.POST("/:id/like", r.postLikeHandler.LikePost, r.authMiddleware.Auth())
		posts.DELETE("/:id/like", r.postLikeHandler.UnlikePost, r.authMiddleware.Auth())
		posts.GET("/:id/likes", r.postLikeHandler.GetPostLikes)
		posts.GET("/:id/like-stats", r.postLikeHandler.GetPostLikeStats)
		posts.GET("/:id/liked", r.postLikeHandler.CheckUserLiked, r.authMiddleware.Auth())
	}
}
