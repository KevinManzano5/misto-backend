// Package cache contains cache
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config struct
type Config struct {
	Addr     string
	Password string
	DB       int
	Timeout  time.Duration
}

// DefaultConfig function
func DefaultConfig() Config {
	return Config{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
		Timeout:  5,
	}
}

// NewClient function
func NewClient(ctx context.Context, config Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         config.Addr,
		Password:     config.Password,
		DB:           config.DB,
		DialTimeout:  config.Timeout,
		ReadTimeout:  config.Timeout,
		WriteTimeout: config.Timeout,
	})

	pingCtx, cancel := context.WithTimeout(ctx, config.Timeout)

	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		return nil, fmt.Errorf("validating redis connection: %w", err)
	}

	return client, nil
}
