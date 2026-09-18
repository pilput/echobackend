package routes

import (
	"echobackend/config"
	"echobackend/internal/handler"
	"echobackend/internal/middleware"
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
	holdingHandler          *handler.HoldingHandler
	exchangeRateHandler     *handler.ExchangeRateHandler
	bookmarkHandler         *handler.BookmarkHandler
	notificationHandler     *handler.NotificationHandler
	reportHandler           *handler.ReportHandler
	corporateActionHandler  *handler.CorporateActionHandler
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
	holdingHandler *handler.HoldingHandler,
	exchangeRateHandler *handler.ExchangeRateHandler,
	bookmarkHandler *handler.BookmarkHandler,
	notificationHandler *handler.NotificationHandler,
	reportHandler *handler.ReportHandler,
	corporateActionHandler *handler.CorporateActionHandler,
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
		holdingHandler:          holdingHandler,
		exchangeRateHandler:     exchangeRateHandler,
		bookmarkHandler:         bookmarkHandler,
		notificationHandler:     notificationHandler,
		reportHandler:           reportHandler,
		corporateActionHandler:  corporateActionHandler,
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
	r.setupHoldingRoutes(api)
	r.setupExchangeRateRoutes(api)
	r.setupBookmarkRoutes(api)
	r.setupNotificationRoutes(api)
	r.setupReportRoutes(api)
}
