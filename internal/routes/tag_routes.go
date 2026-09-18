package routes

import "github.com/labstack/echo/v5"

func (r *Routes) setupTagRoutes(api *echo.Group) {
	tags := api.Group("/tags")
	{
		tags.POST("", r.tagHandler.CreateTag, r.authMiddleware.Auth())
		tags.GET("", r.tagHandler.GetTags)
		tags.GET("/trending", r.tagHandler.GetTrendingTags)
		tags.GET("/sitemap", r.tagHandler.GetTagsForSitemap)
		tags.GET("/:id", r.tagHandler.GetTagByID)
		tags.PUT("/:id", r.tagHandler.UpdateTag, r.authMiddleware.Auth(), r.authMiddleware.AuthAdmin())
		tags.DELETE("/:id", r.tagHandler.DeleteTag, r.authMiddleware.Auth(), r.authMiddleware.AuthAdmin())
	}
}
