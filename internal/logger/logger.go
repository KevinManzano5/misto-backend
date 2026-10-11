// Package logger contains logger
package logger

import (
	"io"
	"log/slog"
	"os"
)

// Config struct
type Config struct {
	Level  slog.Level
	JSON   bool
	Output io.Writer
}

// DefaultConfig function
func DefaultConfig() Config {
	return Config{
		Level:  slog.LevelDebug,
		JSON:   false,
		Output: os.Stdout,
	}
}

// ProductionConfig function
func ProductionConfig() Config {
	return Config{
		Level:  slog.LevelInfo,
		JSON:   true,
		Output: os.Stdout,
	}
}

// New function
func New(config Config) *slog.Logger {
	out := config.Output

	if out == nil {
		out = os.Stdout
	}

	options := &slog.HandlerOptions{
		Level:     config.Level,
		AddSource: config.Level == slog.LevelDebug,
	}

	var handler slog.Handler

	if config.JSON {
		handler = slog.NewJSONHandler(out, options)
	} else {
		handler = slog.NewTextHandler(out, options)
	}

	return slog.New(handler)
}
