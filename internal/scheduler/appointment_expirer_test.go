package scheduler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"
)

type fakeAppointmentExpirer struct {
	calls int32
	err   error
}

func (f *fakeAppointmentExpirer) ExpireOverdueAppointments(ctx context.Context) (int, error) {
	atomic.AddInt32(&f.calls, 1)
	if f.err != nil {
		return 0, f.err
	}
	return 1, nil
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestStart_RunsSweepOnEachTickUntilCancelled(t *testing.T) {
	fake := &fakeAppointmentExpirer{}
	expirer := NewAppointmentExpirer(fake, 5*time.Millisecond, discardLogger())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	expirer.Start(ctx)

	if atomic.LoadInt32(&fake.calls) == 0 {
		t.Fatalf("expected at least one sweep to run before ctx cancellation, got 0")
	}
}

func TestStart_ContinuesAfterSweepError(t *testing.T) {
	fake := &fakeAppointmentExpirer{err: errors.New("boom")}
	expirer := NewAppointmentExpirer(fake, 5*time.Millisecond, discardLogger())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	expirer.Start(ctx)

	if atomic.LoadInt32(&fake.calls) < 2 {
		t.Fatalf("expected the ticker to keep running after a sweep error, got %d calls", fake.calls)
	}
}
