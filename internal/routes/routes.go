package routes

import (
	"echobackend/config"
	"echobackend/internal/handler"
	"echobackend/internal/middleware"
	"echobackend/internal/modules/portfolio"
	"echobackend/internal/platform/cache"

	"github.com/labstack/echo/v5"
)

type Routes struct {
	config                  *config.Config
	cache                   *cache.RedisCache
	userHandler             *handler.UserHandler
	postHandler             *handler.PostHandler
	authHandler             *handler.AuthHandler
	authMiddleware          *middleware.AuthMiddleware
	tagHandler              *handler.TagHandler
	commentHandler          *handler.CommentHandler
	postViewHandler         *handler.PostViewHandler
	postLikeHandler         *handler.PostLikeHandler
	userFollowHandler       *handler.UserFollowHandler
	chatConversationHandler *handler.ChatConversationHandler
	bookmarkHandler         *handler.BookmarkHandler
	notificationHandler     *handler.NotificationHandler
	userStatsHandler        *handler.UserStatsHandler
	postStatsHandler        *handler.PostStatsHandler
	tagStatsHandler         *handler.TagStatsHandler
	postReportHandler       *handler.PostReportHandler
	guildHandler            *handler.GuildHandler
	guildChannelHandler     *handler.GuildChannelHandler
	aiHandler               *handler.AIHandler
	portfolio               *portfolio.Module
}

func NewRoutes(
	config *config.Config,
	redisCache *cache.RedisCache,
	userHandler *handler.UserHandler,
	postHandler *handler.PostHandler,
	authHandler *handler.AuthHandler,
	authMiddleware *middleware.AuthMiddleware,
	tagHandler *handler.TagHandler,
	commentHandler *handler.CommentHandler,
	postViewHandler *handler.PostViewHandler,
	postLikeHandler *handler.PostLikeHandler,
	userFollowHandler *handler.UserFollowHandler,
	chatConversationHandler *handler.ChatConversationHandler,
	bookmarkHandler *handler.BookmarkHandler,
	notificationHandler *handler.NotificationHandler,
	userStatsHandler *handler.UserStatsHandler,
	postStatsHandler *handler.PostStatsHandler,
	tagStatsHandler *handler.TagStatsHandler,
	postReportHandler *handler.PostReportHandler,
	guildHandler *handler.GuildHandler,
	guildChannelHandler *handler.GuildChannelHandler,
	aiHandler *handler.AIHandler,
	portfolioModule *portfolio.Module,
) *Routes {
	return &Routes{
		config:                  config,
		cache:                   redisCache,
		userHandler:             userHandler,
		postHandler:             postHandler,
		authHandler:             authHandler,
		authMiddleware:          authMiddleware,
		tagHandler:              tagHandler,
		commentHandler:          commentHandler,
		postViewHandler:         postViewHandler,
		postLikeHandler:         postLikeHandler,
		userFollowHandler:       userFollowHandler,
		chatConversationHandler: chatConversationHandler,
		bookmarkHandler:         bookmarkHandler,
		notificationHandler:     notificationHandler,
		userStatsHandler:        userStatsHandler,
		postStatsHandler:        postStatsHandler,
		tagStatsHandler:         tagStatsHandler,
		postReportHandler:       postReportHandler,
		guildHandler:            guildHandler,
		guildChannelHandler:     guildChannelHandler,
		aiHandler:               aiHandler,
		portfolio:               portfolioModule,
	}
}

func (r *Routes) Setup(e *echo.Echo) {
	// API Group
	api := e.Group("/api")
	r.setupAPIRoutes(api)
}

func (r *Routes) setupAPIRoutes(api *echo.Group) {
	r.setupUserRoutes(api)
	r.setupPostRoutes(api)
	r.setupAuthRoutes(api)
	r.setupTagRoutes(api)
	r.setupChatConversationRoutes(api)
	r.setupBookmarkRoutes(api)
	r.setupNotificationRoutes(api)
	r.setupGuildRoutes(api)
	r.setupAIRoutes(api)

	// Self-contained modules (internal/modules/*).
	r.portfolio.RegisterRoutes(api, r.authMiddleware.Auth())
}
