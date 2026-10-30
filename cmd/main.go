package main

import (
	"context"
	"echobackend/config"
	"echobackend/internal/di"
	"echobackend/internal/middleware"
	"echobackend/pkg/applog"
	"echobackend/pkg/response"
	"echobackend/pkg/validator"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"
)

func main() {
	applog.SetupFromEnv()

	conf, errconf := config.Load()
	if errconf != nil {
		slog.Error("failed to load config", "error", errconf)
		panic(errconf)
	}

	applog.Setup(conf.App.Debug)

	// Initialize dependency container
	container, err := di.NewContainer(conf)
	if err != nil {
		slog.Error("failed to initialize container", "error", err)
		panic(err)
	}

	e := echo.NewWithConfig(echo.Config{
		Logger: slog.Default(),
		// Router 404/405, body-limit 413 and unhandled errors use the shared
		// {success,message,error} envelope instead of Echo's {"message"} body.
		HTTPErrorHandler: middleware.HTTPErrorHandler,
		// With HTTP_TRUST_PROXY, X-Forwarded-For is walked right-to-left and
		// the first hop outside loopback/link-local/private/CGNAT ranges is the
		// client IP; otherwise the peer address is used. See
		// middleware.IPExtractor.
		IPExtractor: middleware.IPExtractor(conf.HTTP.TrustProxy),
	})

	// Set custom validator
	e.Validator = validator.NewValidator()

	// Global middleware must be registered before routes so CORS, body limit,
	// secure headers, rate limiting, and recover apply to all endpoints.
	middleware.InitMiddleware(e, conf)

	// Initialize routes with manually wired dependencies
	container.Routes.Setup(e)

	e.GET("/", helloWorld)

	// Health check endpoint — used by Docker HEALTHCHECK and load balancers.
	// Returns 200 when the DB is reachable, 503 otherwise.
	e.GET("/health", func(c *echo.Context) error {
		return healthCheck(c, container)
	})

	// Start server in a goroutine.
	// ReadTimeout covers the full request body read. For most endpoints 10 s is fine,
	// but file uploads can be slow on poor connections. We raise it to 60 s here and
	// rely on the 10 MB body limit (middleware) to bound abuse.
	// If you need tighter control per-route, wrap individual handlers with a context
	// deadline instead of changing the global server timeout.
	server := &http.Server{
		Addr:              ":" + conf.HTTP.Port,
		Handler:           e,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	// Long-lived SSE streams (guild chat) would hold Shutdown until its
	// timeout; end them as soon as shutdown begins.
	server.RegisterOnShutdown(container.CloseStreams)

	// A listener failure (e.g. port already in use) must stop the process;
	// otherwise it would keep waiting for a signal while serving nothing.
	serverErr := make(chan error, 1)
	go func() {
		slog.Info("starting server", "port", conf.HTTP.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Wait for an interrupt signal or a server failure, then shut down with a timeout of 10 seconds.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	exitCode := 0
	select {
	case <-quit:
		slog.Info("server is shutting down")
	case err := <-serverErr:
		slog.Error("server exited unexpectedly", "error", err)
		exitCode = 1
	}

	shutdown(server, container)
	os.Exit(exitCode)
}

// shutdown gracefully stops the HTTP server and releases container resources.
func shutdown(server *http.Server, container *di.Container) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	// Cleanup resources
	cleanup, err := di.GetCleanupManager(container)
	if err != nil {
		slog.Error("failed to get cleanup manager", "error", err)
	} else if cleanup != nil {
		if err := cleanup.CleanupWithTimeout(5 * time.Second); err != nil {
			slog.Error("cleanup failed", "error", err)
		} else {
			slog.Info("resources cleaned up successfully")
		}
	}

	slog.Info("server exited")
}

func helloWorld(c *echo.Context) error {
	return response.Success(c, "Hello, World!", nil)
}

// healthCheck pings the database and returns 200 OK or 503 Service Unavailable.
func healthCheck(c *echo.Context, container *di.Container) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 3*time.Second)
	defer cancel()

	if err := container.PingDB(ctx); err != nil {
		slog.Warn("health check: database unreachable", "error", err)
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"status": "unhealthy",
			"reason": "database unreachable",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
	})
}
