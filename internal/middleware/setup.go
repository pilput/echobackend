package middleware

import (
	"echobackend/config"
	"echobackend/pkg/applog"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

var log = applog.Component("http")

// InitMiddleware registers global middleware. Order matters — each entry wraps
// everything registered after it:
//
//  1. Request logger: outermost, so every response (including recovered panics,
//     413 and 429) is logged with its final status.
//  2. Recover: catches panics in all later middleware and handlers.
//  3. CORS: answers preflight requests before they reach the rate limiter, and
//     sets CORS headers before inner middleware can reject the request, so
//     browsers can read 413/429 error bodies instead of reporting a CORS error.
//  4. Secure headers, body limit, rate limit.
func InitMiddleware(e *echo.Echo, config *config.Config) {
	// Enhanced request logging with structured format
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:     true,
		LogStatus:  true,
		LogMethod:  true,
		LogLatency: true,
		LogValuesFunc: func(c *echo.Context, values middleware.RequestLoggerValues) error {
			log.Info("handled request",
				"method", values.Method,
				"uri", values.URI,
				"status", values.Status,
				"latency_ms", float64(values.Latency.Nanoseconds())/1e6,
				"remote_ip", c.RealIP(),
			)
			return nil
		},
	}))

	e.Use(RecoverWithLog())

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{AllowOrigins: config.HTTP.AllowOrigins}))

	// Add security headers
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "SAMEORIGIN",
		HSTSMaxAge:            3600,
		ContentSecurityPolicy: "default-src 'self'",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
	}))

	// Add body limit middleware to prevent memory exhaustion
	e.Use(middleware.BodyLimit(10 * 1024 * 1024)) // Limit request body to 10MB

	// Global HTTP rate limit (sustained RPS, token bucket; 0 = disabled)
	if config.HTTP.RateLimitRPS > 0 {
		storeCfg := middleware.RateLimiterMemoryStoreConfig{
			Rate:  float64(config.HTTP.RateLimitRPS),
			Burst: config.HTTP.RateLimitRPS * 2,
		}
		if config.HTTP.RateLimitWindow > 0 {
			storeCfg.ExpiresIn = config.HTTP.RateLimitWindow
		}
		e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStoreWithConfig(storeCfg)))
	}
}
