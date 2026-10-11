// Package postgres contains db configurations
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config struct contains db configuration
type Config struct {
	URL             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// NewPool function create and validates a connection pool
func NewPool(ctx context.Context, config Config) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(config.URL)

	if err != nil {
		return nil, fmt.Errorf("parsing db config: %w", err)
	}

	poolConfig.MaxConns = config.MaxConns
	poolConfig.MaxConns = config.MinConns
	poolConfig.MaxConnLifetime = config.MaxConnLifetime
	poolConfig.MaxConnIdleTime = config.MaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)

	if err != nil {
		return nil, fmt.Errorf("creating db connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()

		return nil, fmt.Errorf("validating db connection: %w", err)
	}

	return pool, nil
}
