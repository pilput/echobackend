package di

import (
	"context"
	"echobackend/config"
	"echobackend/internal/handler"
	"echobackend/internal/middleware"
	"echobackend/internal/platform/cache"
	"echobackend/internal/platform/database"
	"echobackend/internal/platform/email"
	"echobackend/internal/platform/market"
	"echobackend/internal/platform/openrouter"
	"echobackend/internal/platform/realtime"
	"echobackend/internal/platform/storage"
	"echobackend/internal/repository"
	"echobackend/internal/routes"
	"echobackend/internal/service"

	"gorm.io/gorm"
)

// Container holds the manually wired application dependencies.
type Container struct {
	Config  *config.Config
	Cleanup *CleanupManager
	Routes  *routes.Routes
	db      *gorm.DB
	hub     *realtime.Hub
}

// NewContainer creates a manually wired application container.
func NewContainer(cfg *config.Config) (*Container, error) {
	cleanup := NewCleanupManager()

	db, err := database.NewDatabase(cfg)
	if err != nil {
		return nil, err
	}
	cleanup.Register(func() error {
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	})

	redisCache := cache.NewRedisCache(cfg)
	if redisCache != nil {
		cleanup.Register(func() error {
			return redisCache.Close()
		})
	}

	// Realtime fan-out for guild chat SSE streams; relays through Redis pub/sub
	// when it is available so events reach streams on every instance.
	hub := realtime.NewHub(redisCache)
	hub.Start()
	cleanup.Register(hub.Close)

	s3Storage := storage.NewS3Storage(cfg)
	emailService := email.NewService(cfg.Email)
	cleanup.Register(func() error {
		return emailService.Close()
	})

	userRepo := repository.NewUserRepository(db)
	postRepo := repository.NewPostRepository(db)
	authRepo := repository.NewAuthRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	tagRepo := repository.NewTagRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	postViewRepo := repository.NewPostViewRepository(db)
	postLikeRepo := repository.NewPostLikeRepository(db)
	userFollowRepo := repository.NewUserFollowRepository(db)
	chatConversationRepo := repository.NewChatConversationRepository(db)
	holdingRepo := repository.NewHoldingRepository(db)
	bookmarkRepo := repository.NewBookmarkRepository(db)
	notificationRepo := repository.NewNotificationRepository(db)
	authActivityLogRepo := repository.NewAuthActivityLogRepository(db)
	passwordResetTokenRepo := repository.NewPasswordResetTokenRepository(db)
	reportRepo := repository.NewReportRepository(db)
	corporateActionRepo := repository.NewCorporateActionRepository(db)
	guildRepo := repository.NewGuildRepository(db)
	guildChannelRepo := repository.NewGuildChannelRepository(db)

	authActivityService := service.NewAuthActivityService(authActivityLogRepo)
	openRouterClient := openrouter.NewClient(cfg.OpenRouter)
	userService := service.NewUserService(userRepo, s3Storage)
	tagService := service.NewTagService(tagRepo, redisCache)
	postService := service.NewPostService(postRepo, tagService, s3Storage, redisCache)
	authService := service.NewAuthService(authRepo, userRepo, sessionRepo, passwordResetTokenRepo, authActivityService, cfg, redisCache, emailService)
	notificationService := service.NewNotificationService(notificationRepo)
	commentService := service.NewCommentService(commentRepo, postRepo, notificationService)
	postViewService := service.NewPostViewService(postViewRepo, postRepo, postLikeRepo, redisCache)
	postLikeService := service.NewPostLikeService(postLikeRepo, postRepo)
	userFollowService := service.NewUserFollowService(userFollowRepo, userRepo, notificationService)
	chatConversationService := service.NewChatConversationService(chatConversationRepo, openRouterClient, cfg)

	// Market quotes (RapidAPI with shared Redis caching decorator)
	rawQuoteClient := market.NewRapidAPIQuoteClient(cfg.MarketData.RapidAPIKey, nil)
	quoteClient := market.NewCachedQuoteClient(rawQuoteClient, redisCache)

	holdingService := service.NewHoldingService(holdingRepo, quoteClient, redisCache)
	exchangeRateService := service.NewExchangeRateService(quoteClient, redisCache)
	bookmarkService := service.NewBookmarkService(bookmarkRepo, postRepo)
	reportService := service.NewReportService(reportRepo)
	guildService := service.NewGuildService(guildRepo)
	guildChannelService := service.NewGuildChannelService(guildChannelRepo, hub)

	// Corporate actions: IDX
	idxCorporateClient := market.NewRapidAPIIDXClient(cfg.MarketData.RapidAPIKey, nil)
	corporateActionService := service.NewCorporateActionService(idxCorporateClient, corporateActionRepo)

	userHandler := handler.NewUserHandler(userService, userFollowService)
	postHandler := handler.NewPostHandler(postService, postViewService)
	authHandler := handler.NewAuthHandler(authService, authActivityService, cfg.Frontend)
	tagHandler := handler.NewTagHandler(tagService)
	commentHandler := handler.NewCommentHandler(commentService, userService)
	postViewHandler := handler.NewPostViewHandler(postViewService, userService)
	postLikeHandler := handler.NewPostLikeHandler(postLikeService)
	userFollowHandler := handler.NewUserFollowHandler(userFollowService)
	chatConversationHandler := handler.NewChatConversationHandler(chatConversationService)
	holdingHandler := handler.NewHoldingHandler(holdingService)
	exchangeRateHandler := handler.NewExchangeRateHandler(exchangeRateService)
	bookmarkHandler := handler.NewBookmarkHandler(bookmarkService)
	notificationHandler := handler.NewNotificationHandler(notificationService)
	reportHandler := handler.NewReportHandler(reportService)
	corporateActionHandler := handler.NewCorporateActionHandler(corporateActionService)
	guildHandler := handler.NewGuildHandler(guildService)
	guildChannelHandler := handler.NewGuildChannelHandler(guildChannelService)

	authMiddleware := middleware.NewAuthMiddleware(cfg, userService)
	appRoutes := routes.NewRoutes(
		cfg,
		redisCache,
		userHandler,
		postHandler,
		authHandler,
		authMiddleware,
		tagHandler,
		commentHandler,
		postViewHandler,
		postLikeHandler,
		userFollowHandler,
		chatConversationHandler,
		holdingHandler,
		exchangeRateHandler,
		bookmarkHandler,
		notificationHandler,
		reportHandler,
		corporateActionHandler,
		guildHandler,
		guildChannelHandler,
	)

	return &Container{
		Config:  cfg,
		Cleanup: cleanup,
		Routes:  appRoutes,
		db:      db,
		hub:     hub,
	}, nil
}

// CloseStreams ends every open realtime (SSE) stream. Register it with
// http.Server.RegisterOnShutdown: Shutdown waits for active handlers, and a
// stream otherwise never returns on its own.
func (c *Container) CloseStreams() {
	if c == nil || c.hub == nil {
		return
	}
	_ = c.hub.Close()
}

// PingDB checks that the database connection is alive.
func (c *Container) PingDB(ctx context.Context) error {
	sqlDB, err := c.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// GetCleanupManager retrieves the cleanup manager from the container.
func GetCleanupManager(container *Container) (*CleanupManager, error) {
	if container == nil {
		return nil, nil
	}
	return container.Cleanup, nil
}
