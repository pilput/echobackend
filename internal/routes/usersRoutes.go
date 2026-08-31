package routes

import "github.com/labstack/echo/v5"

func (r *Routes) setupUserRoutes(api *echo.Group) {
	users := api.Group("/users")
	{
		// Public routes
		users.GET("/username/:username", r.userHandler.GetByUsername)

		// Authenticated routes
		authUsers := users.Group("", r.authMiddleware.Auth())
		{
			authUsers.GET("/me", r.userHandler.GetMe)
			authUsers.POST("/me/image", r.userHandler.UploadAvatar)
			authUsers.GET("", r.userHandler.GetUsers, r.authMiddleware.AuthAdmin())
			authUsers.POST("", r.userHandler.CreateUser, r.authMiddleware.AuthAdmin())
			authUsers.GET("/:id", r.userHandler.GetByID, r.authMiddleware.AuthAdmin())
			authUsers.PUT("/:id", r.userHandler.UpdateUser, r.authMiddleware.AuthAdmin())
			authUsers.DELETE("/:id", r.userHandler.DeleteUser, r.authMiddleware.AuthAdmin())
			authUsers.POST("/:id/restore", r.userHandler.RestoreUser, r.authMiddleware.AuthAdmin())

			// Follow routes
			authUsers.POST("/follow", r.userFollowHandler.FollowUser)
			authUsers.DELETE("/:id/follow", r.userFollowHandler.UnfollowUser)
			authUsers.GET("/:id/follow-status", r.userFollowHandler.CheckFollowStatus)
			authUsers.GET("/:id/mutual-follows", r.userFollowHandler.GetMutualFollows)
		}

		// Follow-related public routes
		users.GET("/:id/followers", r.userFollowHandler.GetFollowers)
		users.GET("/:id/following", r.userFollowHandler.GetFollowing)
		users.GET("/:id/follow-stats", r.userFollowHandler.GetFollowStats)
	}
}
