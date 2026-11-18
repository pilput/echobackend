package di

import (
	"context"
	"echobackend/config"
	"echobackend/internal/handler"
	"echobackend/internal/middleware"
	"echobackend/internal/modules/aichat"
	"echobackend/internal/modules/blog"
	"echobackend/internal/modules/guilds"
	"echobackend/internal/modules/portfolio"
	"echobackend/internal/platform/cache"
	"echobackend/internal/platform/database"
	"echobackend/internal/platform/email"
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
	authRepo := repository.NewAuthRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	userFollowRepo := repository.NewUserFollowRepository(db)
	notificationRepo := repository.NewNotificationRepository(db)
	authActivityLogRepo := repository.NewAuthActivityLogRepository(db)
	passwordResetTokenRepo := repository.NewPasswordResetTokenRepository(db)
	accountDeletionTokenRepo := repository.NewAccountDeletionTokenRepository(db)
	userStatsRepo := repository.NewUserStatsRepository(db)

	authActivityService := service.NewAuthActivityService(authActivityLogRepo)
	userService := service.NewUserService(userRepo, s3Storage)
	authService := service.NewAuthService(authRepo, userRepo, sessionRepo, passwordResetTokenRepo, accountDeletionTokenRepo, authActivityService, cfg, redisCache, emailService)
	notificationService := service.NewNotificationService(notificationRepo)
	userFollowService := service.NewUserFollowService(userFollowRepo, userRepo, notificationService)
	userStatsService := service.NewUserStatsService(userStatsRepo)

	userHandler := handler.NewUserHandler(userService, userFollowService)
	authHandler := handler.NewAuthHandler(authService, authActivityService, cfg.Frontend)
	userFollowHandler := handler.NewUserFollowHandler(userFollowService)
	notificationHandler := handler.NewNotificationHandler(notificationService)
	userStatsHandler := handler.NewUserStatsHandler(userStatsService)

	portfolioModule := portfolio.New(portfolio.Deps{
		DB:          db,
		Cache:       redisCache,
		RapidAPIKey: cfg.MarketData.RapidAPIKey,
	})

	aichatModule := aichat.New(aichat.Deps{
		Config: cfg,
		DB:     db,
		Cache:  redisCache,
	})

	guildModule := guilds.New(guilds.Deps{DB: db, Hub: hub})

	blogModule := blog.New(blog.Deps{
		DB:       db,
		Cache:    redisCache,
		Storage:  s3Storage,
		Admins:   adminChecker{users: userService},
		Notifier: notifier{notifications: notificationService},
	})

	authMiddleware := middleware.NewAuthMiddleware(cfg, userService)
	appRoutes := routes.NewRoutes(
		cfg,
		redisCache,
		userHandler,
		authHandler,
		authMiddleware,
		userFollowHandler,
		notificationHandler,
		userStatsHandler,
		portfolioModule,
		aichatModule,
		guildModule,
		blogModule,
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
