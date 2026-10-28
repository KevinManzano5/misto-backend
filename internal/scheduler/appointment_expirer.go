// Package scheduler runs periodic background jobs alongside the API process.
package scheduler

import (
	"context"
	"log/slog"
	"time"
)

// appointmentExpirer is the narrow usecase dependency AppointmentExpirer needs.
type appointmentExpirer interface {
	ExpireOverdueAppointments(ctx context.Context) (int, error)
}

// AppointmentExpirer periodically moves overdue pending/confirmed appointments to a terminal
// status.
type AppointmentExpirer struct {
	usecase  appointmentExpirer
	interval time.Duration
	logger   *slog.Logger
}

// NewAppointmentExpirer constructor.
func NewAppointmentExpirer(usecase appointmentExpirer, interval time.Duration, logger *slog.Logger) *AppointmentExpirer {
	return &AppointmentExpirer{
		usecase:  usecase,
		interval: interval,
		logger:   logger,
	}
}

// Start runs the expiration sweep every interval until ctx is cancelled. It blocks, so callers
// should run it in its own goroutine.
func (expirer *AppointmentExpirer) Start(ctx context.Context) {
	ticker := time.NewTicker(expirer.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			expirer.sweep(ctx)
		}
	}
}

// sweep runs a single expiration pass and logs the outcome.
func (expirer *AppointmentExpirer) sweep(ctx context.Context) {
	count, err := expirer.usecase.ExpireOverdueAppointments(ctx)
	if err != nil {
		expirer.logger.Error("expiring overdue appointments", "error", err)
		return
	}

	if count > 0 {
		expirer.logger.Info("expired overdue appointments", "count", count)
	}
}
