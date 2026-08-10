// Package config loads and validates the application configuration.
//
// All values come from environment variables (a .env file is loaded first if
// present) and are grouped into section configs on the root Config struct:
//
//	cfg.App       // app-level toggles (debug)
//	cfg.HTTP      // listener, rate limit, CORS, proxy trust
//	cfg.Auth      // JWT secret
//	cfg.Database  // PostgreSQL DSN and pool tuning
//	cfg.S3        // S3-compatible object storage
//	cfg.Cache     // Valkey/Redis cache
//	cfg.Queue     // background jobs
//	cfg.Email     // password reset email delivery
//
// Some env keys have fallback aliases (legacy names). The first-set key wins;
// see Load() for the full list.
package config

import (
	"errors"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config is the root application configuration.
type Config struct {
	App        AppConfig
	HTTP       HTTPConfig
	Auth       AuthConfig
	Database   DatabaseConfig
	S3         S3Config
	Cache      CacheConfig
	Queue      QueueConfig
	OpenRouter OpenRouterConfig
	GitHub     GitHubConfig
	Frontend   FrontendConfig
	Email      EmailConfig
	MarketData MarketDataConfig
}

// AppConfig contains application-level toggles.
type AppConfig struct {
	// Debug enables verbose logging, GORM info logs, and debug routes (/api/debug/pprof/*).
	Debug bool
}

// HTTPConfig controls the HTTP server, rate limiting, CORS and proxy trust.
type HTTPConfig struct {
	// Port is the TCP port the HTTP server listens on (e.g. "8080").
	Port string
	// RateLimitRPS is the sustained rate in requests per second for the global
	// Echo rate limiter (0 = disabled). Burst is set to 2x this value.
	RateLimitRPS int
	// RateLimitWindow, when > 0, sets the Echo memory-store visitor ExpiresIn.
	// When 0, Echo defaults (3 minutes) apply.
	RateLimitWindow time.Duration
	// TrustProxy, when true, extracts client IP from X-Forwarded-For.
	// Use only behind a trusted reverse proxy.
	TrustProxy bool
	// AllowOrigins is the parsed CORS allow-list (always non-empty; defaults to ["*"]).
	AllowOrigins []string
}

// AuthConfig contains authentication secrets.
type AuthConfig struct {
	// JWTSecret is the secret key used for JWT token signing and verification.
	JWTSecret string
	// JWTExpiry is the duration for which JWT access tokens remain valid.
	JWTExpiry time.Duration
	// RefreshTokenExpiry is the duration for which refresh tokens remain valid.
	RefreshTokenExpiry time.Duration
}

// DatabaseConfig contains the PostgreSQL DSN and connection pool tuning.
type DatabaseConfig struct {
	// DSN is the PostgreSQL connection string (pgx / GORM DSN).
	DSN string
	// MaxOpenConns is the maximum number of open database connections in the pool.
	MaxOpenConns int
	// MaxIdleConns is the maximum number of idle database connections in the pool.
	MaxIdleConns int
	// ConnMaxLifetime is the maximum duration a database connection can be reused.
	ConnMaxLifetime time.Duration
}

// S3Config contains S3-compatible object storage settings (MinIO, AWS S3, etc.).
type S3Config struct {
	// Endpoint is the S3 service endpoint (e.g. "localhost:9000").
	Endpoint string
	// AccessKey is the S3 access key for authentication.
	AccessKey string
	// SecretKey is the S3 secret key for authentication.
	SecretKey string
	// Bucket is the default S3 bucket name.
	Bucket string
	// UseSSL determines whether to use SSL/TLS for S3 connections.
	UseSSL bool
}

// CacheConfig contains Redis/Valkey cache settings.
type CacheConfig struct {
	// RedisURL is the connection URL for Redis/Valkey (e.g. redis://localhost:6379/0).
	// Empty disables caching (the app runs fail-open).
	RedisURL string
	// KeyPrefix is prepended to cache keys to avoid collisions across apps/environments.
	KeyPrefix string
	// TTL is the default cache lifetime. 0 effectively disables writes.
	TTL time.Duration
	// ConnectTimeout is used when establishing a new cache connection.
	ConnectTimeout time.Duration
}

// QueueConfig contains Asynq background job settings.
type QueueConfig struct {
	// RedisURL is the connection URL for the Asynq broker. Empty disables background jobs.
	RedisURL string
	// ConnectTimeout is used for Redis dial/read/write timeouts.
	ConnectTimeout time.Duration
	// DefaultQueue is used when callers do not specify a queue.
	DefaultQueue string
	// Concurrency controls the number of concurrent Asynq workers.
	Concurrency int
	// MaxRetry is the default retry count for queued tasks.
	MaxRetry int
}

// OpenRouterConfig contains upstream AI chat settings.
type OpenRouterConfig struct {
	APIKey       string
	BaseURL      string
	DefaultModel string
	HTTPReferer  string
	Title        string
	Timeout      time.Duration
}

// GitHubConfig contains GitHub OAuth settings.
type GitHubConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

// FrontendConfig contains frontend URL settings for OAuth redirects and cookies.
type FrontendConfig struct {
	URL              string
	OAuthCallbackURL string
	ResetPasswordURL string
	MainDomain       string
}

// EmailConfig contains email delivery settings.
type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	From         string
	Timeout      time.Duration
	TaskTimeout  time.Duration
	UseTLS       bool
}

// MarketDataConfig contains API keys for external financial data providers.
type MarketDataConfig struct {
	// RapidAPIIDXKey is the X-RapidAPI-Key for the Indonesia Stock Exchange API.
	// Required to fetch IDX dividend and RUPS calendar data.
	// Leave empty to disable IDX corporate-action fetching (returns empty results).
	RapidAPIIDXKey string
}

// Load reads configuration from environment variables with defaults.
//
// It loads a .env file if present, then reads environment variables.
// Returns a validated *Config or an error if required fields are missing.
//
// Some keys accept legacy aliases; the first-set key wins.
func Load() (*Config, error) {
	// Best-effort .env loading; missing file is not an error.
	_ = godotenv.Load()

	cfg := &Config{
		App: AppConfig{
			Debug: envBool([]string{"APP_DEBUG", "DEBUG"}, false),
		},
		HTTP: HTTPConfig{
			Port:            envString([]string{"PORT"}, "8080"),
			RateLimitRPS:    envInt([]string{"HTTP_RATE_LIMIT_RPS", "RATE_LIMITER_MAX"}, 0),
			RateLimitWindow: time.Duration(envInt([]string{"HTTP_RATE_LIMIT_WINDOW_SEC", "RATE_LIMITER_TTL"}, 0)) * time.Second,
			TrustProxy:      envBool([]string{"HTTP_TRUST_PROXY", "TRUST_PROXY"}, false),
			AllowOrigins:    parseOrigins(envString([]string{"HTTP_ALLOW_ORIGINS"}, "*")),
		},
		Auth: AuthConfig{
			JWTSecret:          envString([]string{"JWT_SECRET"}, ""),
			JWTExpiry:          time.Duration(envInt([]string{"JWT_EXPIRY_HOURS"}, 3)) * time.Hour,
			RefreshTokenExpiry: time.Duration(envInt([]string{"REFRESH_TOKEN_EXPIRY_DAYS"}, 7)) * 24 * time.Hour,
		},
		Database: DatabaseConfig{
			DSN:             envString([]string{"DATABASE_URL"}, ""),
			MaxOpenConns:    envInt([]string{"DB_POOL_MAX_OPEN", "MAX_OPEN_CONNS"}, 25),
			MaxIdleConns:    envInt([]string{"DB_POOL_MAX_IDLE", "MAX_IDLE_CONNS"}, 25),
			ConnMaxLifetime: envDuration([]string{"DB_POOL_CONN_LIFETIME", "CONN_MAX_LIFETIME"}, 15*time.Minute),
		},
		S3: S3Config{
			Endpoint:  envString([]string{"S3_ENDPOINT", "MINIO_ENDPOINT"}, "localhost:9000"),
			AccessKey: envString([]string{"S3_ACCESS_KEY", "MINIO_ACCESS_KEY"}, "minioadmin"),
			SecretKey: envString([]string{"S3_SECRET_KEY", "MINIO_SECRET_KEY"}, "minioadmin"),
			Bucket:    envString([]string{"S3_BUCKET", "MINIO_BUCKET"}, "minio-bucket"),
			UseSSL:    envBool([]string{"S3_USE_SSL", "MINIO_USE_SSL"}, true),
		},
		Cache: CacheConfig{
			RedisURL:       envString([]string{"REDIS_URL", "VALKEY_URL"}, ""),
			KeyPrefix:      envString([]string{"CACHE_KEY_PREFIX"}, "pilput"),
			TTL:            time.Duration(envInt([]string{"CACHE_TTL_SECONDS"}, 60)) * time.Second,
			ConnectTimeout: time.Duration(envInt([]string{"REDIS_CONNECT_TIMEOUT_MS", "VALKEY_CONNECT_TIMEOUT_MS"}, 2000)) * time.Millisecond,
		},
		Queue: QueueConfig{
			RedisURL:       envString([]string{"QUEUE_REDIS_URL", "ASYNQ_REDIS_URL", "REDIS_URL", "VALKEY_URL"}, ""),
			ConnectTimeout: time.Duration(envInt([]string{"QUEUE_REDIS_TIMEOUT_MS", "ASYNQ_REDIS_TIMEOUT_MS", "REDIS_CONNECT_TIMEOUT_MS", "VALKEY_CONNECT_TIMEOUT_MS"}, 2000)) * time.Millisecond,
			DefaultQueue:   envString([]string{"QUEUE_DEFAULT_NAME"}, "default"),
			Concurrency:    envInt([]string{"QUEUE_CONCURRENCY"}, 1),
			MaxRetry:       envInt([]string{"QUEUE_MAX_RETRY"}, 5),
		},
		OpenRouter: OpenRouterConfig{
			APIKey:       envString([]string{"OPENROUTER_API_KEY"}, ""),
			BaseURL:      envString([]string{"OPENROUTER_BASE_URL"}, "https://openrouter.ai/api/v1"),
			DefaultModel: envString([]string{"OPENROUTER_DEFAULT_MODEL"}, "openrouter/free"),
			HTTPReferer:  envString([]string{"OPENROUTER_HTTP_REFERER"}, "https://pilput.net"),
			Title:        envString([]string{"OPENROUTER_TITLE"}, "pilput"),
			Timeout:      time.Duration(envInt([]string{"OPENROUTER_TIMEOUT_SECONDS"}, 90)) * time.Second,
		},
		GitHub: GitHubConfig{
			ClientID:     envString([]string{"GITHUB_CLIENT_ID"}, ""),
			ClientSecret: envString([]string{"GITHUB_CLIENT_SECRET"}, ""),
			RedirectURI:  envString([]string{"GITHUB_REDIRECT_URI"}, "http://localhost:8080/api/auth/oauth/github/callback"),
		},
		Frontend: FrontendConfig{
			URL:              envString([]string{"FRONTEND_URL"}, "http://localhost:3000"),
			OAuthCallbackURL: envString([]string{"FRONTEND_OAUTH_CALLBACK_URL"}, "http://localhost:3000/auth/callback"),
			ResetPasswordURL: envString([]string{"FRONTEND_RESET_PASSWORD_URL"}, "http://localhost:3000/reset-password"),
			MainDomain:       envString([]string{"MAIN_DOMAIN"}, "localhost"),
		},
		Email: EmailConfig{
			SMTPHost:     envString([]string{"SMTP_HOST"}, ""),
			SMTPPort:     envInt([]string{"SMTP_PORT"}, 587),
			SMTPUsername: envString([]string{"SMTP_USERNAME"}, ""),
			SMTPPassword: envString([]string{"SMTP_PASSWORD"}, ""),
			From:         envString([]string{"SMTP_FROM", "EMAIL_FROM"}, "noreply@pilput.net"),
			Timeout:      time.Duration(envInt([]string{"SMTP_TIMEOUT_SECONDS"}, 10)) * time.Second,
			TaskTimeout:  time.Duration(envInt([]string{"SMTP_TASK_TIMEOUT_SECONDS"}, 30)) * time.Second,
			UseTLS:       envBool([]string{"SMTP_TLS"}, false),
		},
		MarketData: MarketDataConfig{
			RapidAPIIDXKey: envString([]string{"RAPIDAPI_IDX_KEY"}, ""),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate checks cross-section invariants and required fields.
func (c *Config) validate() error {
	if c.Auth.JWTSecret == "" {
		return errors.New("JWT_SECRET is required")
	}
	if len(c.Auth.JWTSecret) < 32 {
		return errors.New("JWT_SECRET must be at least 32 characters long")
	}
	if c.Auth.JWTExpiry <= 0 {
		return errors.New("JWT_EXPIRY_HOURS must be > 0")
	}
	if c.Auth.RefreshTokenExpiry <= 0 {
		return errors.New("REFRESH_TOKEN_EXPIRY_DAYS must be > 0")
	}
	if c.Database.DSN == "" {
		return errors.New("DATABASE_URL is required")
	}
	if c.Cache.TTL < 0 {
		return errors.New("CACHE_TTL_SECONDS must be >= 0")
	}
	if c.Cache.ConnectTimeout < 0 {
		return errors.New("REDIS_CONNECT_TIMEOUT_MS must be >= 0")
	}
	if c.Queue.ConnectTimeout <= 0 {
		return errors.New("QUEUE_REDIS_TIMEOUT_MS must be > 0")
	}
	if c.Queue.DefaultQueue == "" {
		return errors.New("QUEUE_DEFAULT_NAME is required")
	}
	if c.Queue.Concurrency <= 0 {
		return errors.New("QUEUE_CONCURRENCY must be > 0")
	}
	if c.Queue.MaxRetry < 0 {
		return errors.New("QUEUE_MAX_RETRY must be >= 0")
	}
	if c.HTTP.RateLimitRPS < 0 {
		return errors.New("HTTP_RATE_LIMIT_RPS must be >= 0")
	}
	if c.Email.SMTPPort <= 0 || c.Email.SMTPPort > 65535 {
		return errors.New("SMTP_PORT must be between 1 and 65535")
	}
	if c.Email.Timeout <= 0 {
		return errors.New("SMTP_TIMEOUT_SECONDS must be > 0")
	}
	if c.Email.TaskTimeout <= 0 {
		return errors.New("SMTP_TASK_TIMEOUT_SECONDS must be > 0")
	}
	return nil
}

// parseOrigins splits a comma-separated CORS origin list, trims whitespace,
// and drops empty entries. Returns ["*"] when the raw value is empty or "*".
func parseOrigins(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "*" {
		return []string{"*"}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return []string{"*"}
	}
	return out
}
