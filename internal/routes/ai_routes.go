package routes

import (
	"time"

	appmiddleware "echobackend/internal/middleware"

	"github.com/labstack/echo/v5"
)

func (r *Routes) setupAIRoutes(api *echo.Group) {
	generateArticleRateLimit := appmiddleware.FixedWindowRateLimiterWithCache(r.cache, "ai:generate-article", 20, time.Hour)

	ai := api.Group("/ai")
	// Super admins only for now (rollout); privileges are read from the DB.
	ai.POST("/generate-article", r.aiHandler.GenerateArticle, r.authMiddleware.Auth(), r.authMiddleware.AuthAdmin(), generateArticleRateLimit)
}
