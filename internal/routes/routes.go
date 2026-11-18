package routes

import (
	"echobackend/config"
	"echobackend/internal/handler"
	"echobackend/internal/middleware"
	"echobackend/internal/modules/aichat"
	"echobackend/internal/modules/blog"
	"echobackend/internal/modules/guilds"
	"echobackend/internal/modules/portfolio"
	"echobackend/internal/platform/cache"

	"github.com/labstack/echo/v5"
)

type Routes struct {
	config              *config.Config
	cache               *cache.RedisCache
	userHandler         *handler.UserHandler
	authHandler         *handler.AuthHandler
	authMiddleware      *middleware.AuthMiddleware
	userFollowHandler   *handler.UserFollowHandler
	notificationHandler *handler.NotificationHandler
	userStatsHandler    *handler.UserStatsHandler
	portfolio           *portfolio.Module
	aichat              *aichat.Module
	guild               *guilds.Module
	blog                *blog.Module
}

func NewRoutes(
	config *config.Config,
	redisCache *cache.RedisCache,
	userHandler *handler.UserHandler,
	authHandler *handler.AuthHandler,
	authMiddleware *middleware.AuthMiddleware,
	userFollowHandler *handler.UserFollowHandler,
	notificationHandler *handler.NotificationHandler,
	userStatsHandler *handler.UserStatsHandler,
	portfolioModule *portfolio.Module,
	aichatModule *aichat.Module,
	guildModule *guilds.Module,
	blogModule *blog.Module,
) *Routes {
	return &Routes{
		config:              config,
		cache:               redisCache,
		userHandler:         userHandler,
		authHandler:         authHandler,
		authMiddleware:      authMiddleware,
		userFollowHandler:   userFollowHandler,
		notificationHandler: notificationHandler,
		userStatsHandler:    userStatsHandler,
		portfolio:           portfolioModule,
		aichat:              aichatModule,
		guild:               guildModule,
		blog:                blogModule,
	}
}

func (r *Routes) Setup(e *echo.Echo) {
	// API Group
	api := e.Group("/api")
	r.setupAPIRoutes(api)
}

func (r *Routes) setupAPIRoutes(api *echo.Group) {
	r.setupUserRoutes(api)
	r.setupAuthRoutes(api)
	r.setupNotificationRoutes(api)

	// Self-contained modules (internal/modules/*).
	r.portfolio.RegisterRoutes(api, r.authMiddleware.Auth())
	r.aichat.RegisterRoutes(api, aichat.Guards{Auth: r.authMiddleware.Auth(), Admin: r.authMiddleware.AuthAdmin()})
	r.guild.RegisterRoutes(api, guilds.Guards{Auth: r.authMiddleware.Auth(), OptionalAuth: r.authMiddleware.OptionalAuth()})
	r.blog.RegisterRoutes(api, blog.Guards{Auth: r.authMiddleware.Auth(), OptionalAuth: r.authMiddleware.OptionalAuth(), Admin: r.authMiddleware.AuthAdmin()})
}
