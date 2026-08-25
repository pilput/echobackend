package di

import (
	"context"
	"echobackend/config"
	"echobackend/internal/handler"
	"echobackend/internal/middleware"
	"echobackend/internal/platform/cache"
	"echobackend/internal/platform/database"
	"echobackend/internal/platform/email"
	"echobackend/internal/platform/queue"
	"echobackend/internal/platform/storage"
	"echobackend/internal/repository"
	"echobackend/internal/routes"
	"echobackend/internal/service"
	"echobackend/pkg/market"

	"gorm.io/gorm"
)

// Container holds the manually wired application dependencies.
type Container struct {
	Config  *config.Config
	Cleanup *CleanupManager
	Routes  *routes.Routes
	db      *gorm.DB
}

// NewContainer creates a manually wired application container.
func NewContainer(cfg *config.Config) (*Container, error) {
	cleanup := NewCleanupManager()

	db := database.NewDatabase(cfg)
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

	taskQueue := queue.NewService(cfg.Queue)
	cleanup.Register(func() error {
		return taskQueue.Close()
	})

	s3Storage := storage.NewS3Storage(cfg)
	emailService := email.NewService(cfg.Email, taskQueue)
	cleanup.Register(func() error {
		return emailService.Close()
	})
	taskQueue.Start()

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

	authActivityService := service.NewAuthActivityService(authActivityLogRepo)
	openRouterService := service.NewOpenRouterService(cfg.OpenRouter)
	userService := service.NewUserService(userRepo)
	tagService := service.NewTagService(tagRepo, redisCache)
	postService := service.NewPostService(postRepo, tagService, s3Storage, redisCache)
	authService := service.NewAuthService(authRepo, userRepo, sessionRepo, passwordResetTokenRepo, authActivityService, cfg, redisCache, emailService)
	notificationService := service.NewNotificationService(notificationRepo)
	commentService := service.NewCommentService(commentRepo, postRepo, notificationService)
	postViewService := service.NewPostViewService(postViewRepo, postRepo, postLikeRepo)
	postLikeService := service.NewPostLikeService(postLikeRepo, postRepo)
	userFollowService := service.NewUserFollowService(userFollowRepo, userRepo, notificationService)
	chatConversationService := service.NewChatConversationService(chatConversationRepo, openRouterService, cfg)

	// Market quotes (RapidAPI with shared Redis caching decorator)
	rawQuoteClient := market.NewRapidAPIQuoteClient(
		cfg.MarketData.RapidAPIQuoteKey,
		cfg.MarketData.RapidAPIQuoteHost,
		cfg.MarketData.RapidAPIQuoteBaseURL,
		nil,
	)
	quoteClient := market.NewCachedQuoteClient(rawQuoteClient, redisCache, cfg.MarketData.QuoteCacheTTL)

	holdingService := service.NewHoldingService(holdingRepo, quoteClient, redisCache)
	exchangeRateService := service.NewExchangeRateService(quoteClient, redisCache)
	bookmarkService := service.NewBookmarkService(bookmarkRepo, postRepo)
	reportService := service.NewReportService(reportRepo)

	// Corporate actions: IDX
	idxCorporateClient := market.NewRapidAPIIDXClient(cfg.MarketData.RapidAPIIDXKey, nil)
	corporateActionService := service.NewCorporateActionService(idxCorporateClient, corporateActionRepo)

	userHandler := handler.NewUserHandler(userService, userFollowService)
	postHandler := handler.NewPostHandler(postService, postViewService)
	authHandler := handler.NewAuthHandler(authService, authActivityService, cfg.Frontend)
	tagHandler := handler.NewTagHandler(tagService)
	commentHandler := handler.NewCommentHandler(commentService)
	postViewHandler := handler.NewPostViewHandler(postViewService)
	postLikeHandler := handler.NewPostLikeHandler(postLikeService)
	userFollowHandler := handler.NewUserFollowHandler(userFollowService)
	chatConversationHandler := handler.NewChatConversationHandler(chatConversationService)
	holdingHandler := handler.NewHoldingHandler(holdingService)
	exchangeRateHandler := handler.NewExchangeRateHandler(exchangeRateService)
	bookmarkHandler := handler.NewBookmarkHandler(bookmarkService)
	notificationHandler := handler.NewNotificationHandler(notificationService)
	reportHandler := handler.NewReportHandler(reportService)
	corporateActionHandler := handler.NewCorporateActionHandler(corporateActionService)

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
	)

	return &Container{
		Config:  cfg,
		Cleanup: cleanup,
		Routes:  appRoutes,
		db:      db,
	}, nil
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
