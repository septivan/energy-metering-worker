package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Pool is an alias for pgxpool.Pool
type Pool = pgxpool.Pool

// NewPool creates a new PostgreSQL connection pool
func NewPool(lc fx.Lifecycle, logger *zap.Logger, databaseURL string) (*pgxpool.Pool, error) {
	logger.Info("initializing database connection pool")

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("[DATABASE] failed to parse database URL: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("[DATABASE] failed to create connection pool: %w", err)
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("attempting to connect to database...")
			if err := pool.Ping(ctx); err != nil {
				logger.Error("database ping failed", zap.Error(err), zap.String("url", maskPassword(databaseURL)))
				return fmt.Errorf("[DATABASE CONNECTION FAILED] cannot reach database. Please check: 1) Database is running, 2) DATABASE_URL is correct, 3) Network/firewall allows connection. Error: %w", err)
			}
			logger.Info("database connection established successfully")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			pool.Close()
			logger.Info("database connection closed")
			return nil
		},
	})

	return pool, nil
}

// maskPassword masks the password in database URL for logging
func maskPassword(url string) string {
	if len(url) == 0 {
		return "<empty>"
	}
	// Simple masking - find password part between : and @
	start := 0
	for i := 0; i < len(url); i++ {
		if url[i] == ':' && i > 0 && url[i-1] != '/' {
			start = i + 1
		}
		if url[i] == '@' && start > 0 {
			return url[:start] + "***" + url[i:]
		}
	}
	return url
}
