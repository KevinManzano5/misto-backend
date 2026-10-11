package server

import (
	"log/slog"
	"time"
)

type config struct {
	readTimeout       time.Duration
	writeTimeout      time.Duration
	idleTimeout       time.Duration
	readHeaderTimeout time.Duration
	logger            *slog.Logger
}

// defaultConfig function
func defaultConfig() config {
	return config{
		readTimeout:       10 * time.Second,
		writeTimeout:      30 * time.Second,
		idleTimeout:       60 * time.Second,
		readHeaderTimeout: 5 * time.Second,
		logger:            slog.Default(),
	}
}

// Option struct
type Option func(*config)

// WithReadTimeout function
func WithReadTimeout(duration time.Duration) Option {
	return func(c *config) {
		c.readTimeout = duration
	}
}

// WithWriteTimeout function
func WithWriteTimeout(duration time.Duration) Option {
	return func(c *config) {
		c.writeTimeout = duration
	}
}

// WithIdleTimeout function
func WithIdleTimeout(duration time.Duration) Option {
	return func(c *config) {
		c.idleTimeout = duration
	}
}

// WithReadHeaderTimeout function
func WithReadHeaderTimeout(duration time.Duration) Option {
	return func(c *config) {
		c.readHeaderTimeout = duration
	}
}

// WithLogger function
func WithLogger(logger *slog.Logger) Option {
	return func(c *config) {
		c.logger = logger
	}
}
