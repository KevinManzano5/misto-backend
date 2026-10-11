// Package notification contains notifications
package notification

import (
	"context"
	"log/slog"
)

// Notifier interface
type Notifier interface {
	Notify(ctx context.Context, userID string, message string) error
}

// LogNotifier struct
type LogNotifier struct {
	logger *slog.Logger
}

// NewLogNotifier function
func NewLogNotifier(logger *slog.Logger) *LogNotifier {
	return &LogNotifier{
		logger: logger,
	}
}

// Notify method
func (notifier *LogNotifier) Notify(ctx context.Context, userID string, message string) error {
	notifier.logger.InfoContext(ctx, "notification sended",
		slog.String("user_id", userID),
		slog.String("message", message))

	return nil
}

// NoOpNotifier struct
type NoOpNotifier struct{}

// Notify method
func (notifier *NoOpNotifier) Notify(_ context.Context, _ string, _ string) error {
	return nil
}
