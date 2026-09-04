package database

import (
	"database/sql"
	"echobackend/config"
	"echobackend/pkg/applog"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var log = applog.Component("database")

// NewDatabase opens the connection pool and wraps it in GORM.
//
// Connection-level settings belong in DATABASE_URL, not here: pgx parses the
// libpq connection string and validates it, so connect_timeout, sslmode and
// default_query_exec_mode are set the same way every other Postgres tool sets
// them. Only pool sizing — which is a property of this process, not of the
// connection — is configured in code.
//
// Reach for stdlib.OpenDB with a hand-built pgx.ConnConfig only if something is
// ever needed that has no connection-string equivalent, such as a custom
// tls.Config or an AfterConnect hook.
func NewDatabase(config *config.Config) (*gorm.DB, error) {
	gormLogLevel := logger.Error
	if config.App.Debug {
		gormLogLevel = logger.Info
	}

	gormConfig := &gorm.Config{
		Logger: logger.NewSlogLogger(applog.Component("gorm").Slog(), logger.Config{
			LogLevel:                  gormLogLevel,
			SlowThreshold:             200 * time.Millisecond,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      !config.App.Debug,
		}),
		// Turns driver errors into gorm.ErrDuplicatedKey / ErrForeignKeyViolated,
		// so repositories can match on those instead of reaching for pgconn and
		// SQLSTATE strings.
		TranslateError: true,
	}

	// gorm.Open pings before returning, because DisableAutomaticPing is left
	// false and the pool it builds satisfies its Pinger check. Startup therefore
	// blocks until the database answers, and an unreachable database fails the
	// process here instead of surfacing as 500s on the first request. The
	// /health endpoint re-checks liveness afterwards. Keep connect_timeout in
	// the DSN so a hung TCP connect cannot stall startup for the OS default.
	db, err := gorm.Open(postgres.Open(config.Database.DSN), gormConfig)
	if err != nil {
		// GORM closes the pool itself when its own ping fails.
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	sqldb, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to reach underlying sql.DB: %w", err)
	}

	poolConfig := connectionPoolConfig{
		maxOpenConns:    defaultInt(config.Database.MaxOpenConns, 25),
		maxIdleConns:    defaultInt(config.Database.MaxIdleConns, 25),
		connMaxLifetime: defaultDuration(config.Database.ConnMaxLifetime, 15*time.Minute),
		connMaxIdleTime: defaultDuration(config.Database.ConnMaxIdleTime, 5*time.Minute),
	}
	configureConnectionPool(sqldb, poolConfig)

	log.Info("pool ready",
		"max_open", poolConfig.maxOpenConns,
		"max_idle", poolConfig.maxIdleConns,
		"conn_lifetime", poolConfig.connMaxLifetime,
		"conn_idle_time", poolConfig.connMaxIdleTime,
	)
	return db, nil
}

type connectionPoolConfig struct {
	maxOpenConns    int
	maxIdleConns    int
	connMaxLifetime time.Duration
	connMaxIdleTime time.Duration
}

func configureConnectionPool(db *sql.DB, cfg connectionPoolConfig) {
	db.SetMaxOpenConns(cfg.maxOpenConns)
	db.SetMaxIdleConns(cfg.maxIdleConns)
	db.SetConnMaxLifetime(cfg.connMaxLifetime)
	db.SetConnMaxIdleTime(cfg.connMaxIdleTime)
}

func defaultInt(value, fallback int) int {
	if value == 0 {
		return fallback
	}
	return value
}

func defaultDuration(value, fallback time.Duration) time.Duration {
	if value == 0 {
		return fallback
	}
	return value
}
