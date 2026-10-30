package middleware

import (
	"math"
	"strconv"
	"time"

	"echobackend/config"
	"echobackend/pkg/applog"
	"echobackend/pkg/response"

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
//
// Errors returned through this chain (404/405 from the router, 413 from the
// body limit, recovered panics) are rendered by HTTPErrorHandler, which
// cmd/main.go installs as e.HTTPErrorHandler.
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

	// Global HTTP rate limit (per-IP token bucket; 0 = disabled).
	if config.HTTP.RateLimitRPS > 0 {
		e.Use(GlobalRateLimiter(config.HTTP.RateLimitRPS, config.HTTP.RateLimitWindow))
	}
}

// GlobalRateLimiter limits each client IP (c.RealIP()) with an in-memory
// token bucket: rps tokens per second, burst 2*rps. idleExpiry only sets how
// long an idle visitor is remembered (0 = Echo's default 3 minutes); it is not
// a counting window. A denied request gets 429 with the shared envelope and a
// Retry-After header. fiberbackend's GlobalRateLimiter and axumbackend's
// global limiter implement the same contract.
func GlobalRateLimiter(rps int, idleExpiry time.Duration) echo.MiddlewareFunc {
	storeCfg := middleware.RateLimiterMemoryStoreConfig{
		Rate:  float64(rps),
		Burst: rps * 2,
	}
	if idleExpiry > 0 {
		storeCfg.ExpiresIn = idleExpiry
	}
	retryAfter := strconv.Itoa(globalRetryAfterSeconds(rps))

	return middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(storeCfg),
		IdentifierExtractor: func(c *echo.Context) (string, error) {
			return c.RealIP(), nil
		},
		DenyHandler: func(c *echo.Context, _ string, _ error) error {
			c.Response().Header().Set("Retry-After", retryAfter)
			return response.TooManyRequests(c, response.MsgTooManyAttempts)
		},
	})
}

// globalRetryAfterSeconds is the time until a drained bucket refills one
// token (1/rps), rounded up and never below one second.
func globalRetryAfterSeconds(rps int) int {
	if rps <= 0 {
		return 1
	}
	return max(int(math.Ceil(1/float64(rps))), 1)
}
