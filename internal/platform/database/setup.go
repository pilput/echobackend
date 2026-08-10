package database

import (
	"database/sql"
	"echobackend/config"
	"echobackend/pkg/applog"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var log = applog.Component("database")

// NewDatabase creates a new database connection using the provided configuration
func NewDatabase(config *config.Config) *gorm.DB {
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
	}

	pgxConfig, err := pgx.ParseConfig(config.Database.DSN)
	if err != nil {
		log.Error("failed to parse database config", "error", err)
		panic(fmt.Errorf("failed to parse database config: %w", err))
	}
	// Use the default extended query protocol for better performance (named statements,
	// binary encoding). If you run behind PgBouncer in transaction-pooling mode, set
	// PGX_QUERY_EXEC_MODE=simple or switch back to QueryExecModeSimpleProtocol here.
	pgxConfig.ConnectTimeout = 10 * time.Second

	// Open connection pool — connections are established lazily on first use.
	// We intentionally skip a blocking Ping here to keep startup fast;
	// the /health endpoint verifies liveness on demand.
	poolConfig := connectionPoolConfig{
		maxOpenConns:    defaultInt(config.Database.MaxOpenConns, 25),
		maxIdleConns:    defaultInt(config.Database.MaxIdleConns, 25),
		connMaxLifetime: defaultDuration(config.Database.ConnMaxLifetime, 15*time.Minute),
		connMaxIdleTime: 5 * time.Minute,
	}

	sqldb := stdlib.OpenDB(*pgxConfig)
	configureConnectionPool(sqldb, poolConfig)

	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqldb,
	}), gormConfig)
	if err != nil {
		_ = sqldb.Close()
		log.Error("failed to open database", "error", err)
		panic(fmt.Errorf("failed to open database: %w", err))
	}

	log.Info("pool ready", "max_open", poolConfig.maxOpenConns, "max_idle", poolConfig.maxIdleConns, "conn_lifetime", poolConfig.connMaxLifetime)
	return db
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
