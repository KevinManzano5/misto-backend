package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/KevinManzano5/misto-backend/internal/domain"
)

// fakeAppointmentRepository is a minimal AppointmentRepository test double; only the methods
// exercised by a given test need a non-nil func field.
type fakeAppointmentRepository struct {
	expireOverdueAppointmentsFunc func(ctx context.Context, before time.Time) ([]*domain.Appointment, error)
	getAppointmentByIDFunc        func(ctx context.Context, id string) (*domain.Appointment, error)
	updateAppointmentStatusFunc   func(ctx context.Context, id, status string) (*domain.Appointment, error)
	listAppointmentsByClientFunc  func(ctx context.Context, clientID string) ([]*domain.AppointmentWithStore, error)
}

func (f *fakeAppointmentRepository) CreateAppointment(ctx context.Context, appointment *domain.Appointment) error {
	panic("not implemented")
}

func (f *fakeAppointmentRepository) ListAppointmentsByStoreDate(ctx context.Context, storeID string, startTime, endTime time.Time) ([]*domain.Appointment, error) {
	panic("not implemented")
}

func (f *fakeAppointmentRepository) ListAllAppointmentsByStoreWithClient(ctx context.Context, storeID string, startTime, endTime *time.Time) ([]*domain.AppointmentWithClient, error) {
	panic("not implemented")
}

func (f *fakeAppointmentRepository) GetAppointmentByID(ctx context.Context, id string) (*domain.Appointment, error) {
	return f.getAppointmentByIDFunc(ctx, id)
}

func (f *fakeAppointmentRepository) UpdateAppointmentStatus(ctx context.Context, id, status string) (*domain.Appointment, error) {
	return f.updateAppointmentStatusFunc(ctx, id, status)
}

func (f *fakeAppointmentRepository) ExpireOverdueAppointments(ctx context.Context, before time.Time) ([]*domain.Appointment, error) {
	return f.expireOverdueAppointmentsFunc(ctx, before)
}

func (f *fakeAppointmentRepository) ListAppointmentsByClient(ctx context.Context, clientID string) ([]*domain.AppointmentWithStore, error) {
	return f.listAppointmentsByClientFunc(ctx, clientID)
}

// fakeAppointmentStoreRepository is a minimal AppointmentStoreRepository test double.
type fakeAppointmentStoreRepository struct {
	getStoreByIDFunc func(ctx context.Context, storeID string) (*domain.Store, error)
}

func (f *fakeAppointmentStoreRepository) GetStoreByID(ctx context.Context, storeID string) (*domain.Store, error) {
	return f.getStoreByIDFunc(ctx, storeID)
}

func TestExpireOverdueAppointments_ReturnsCountFromRepository(t *testing.T) {
	repo := &fakeAppointmentRepository{
		expireOverdueAppointmentsFunc: func(ctx context.Context, before time.Time) ([]*domain.Appointment, error) {
			return []*domain.Appointment{
				{ID: "1", Status: domain.AppointmentStatusCompleted},
				{ID: "2", Status: domain.AppointmentStatusExpired},
			}, nil
		},
	}
	uc := NewAppointmentUseCase(repo, nil, nil)

	count, err := uc.ExpireOverdueAppointments(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected count 2, got %d", count)
	}
}

func TestParseBookedAppointmentsDateRange(t *testing.T) {
	loc := time.UTC

	tests := []struct {
		name         string
		startDateStr string
		endDateStr   string
		wantErr      bool
	}{
		{
			name: "no filter leaves both bounds open",
		},
		{
			name:         "only start_date leaves end open",
			startDateStr: "2026-01-10",
		},
		{
			name:       "only end_date leaves start open",
			endDateStr: "2026-01-10",
		},
		{
			name:         "invalid start_date is rejected",
			startDateStr: "not-a-date",
			wantErr:      true,
		},
		{
			name:       "invalid end_date is rejected",
			endDateStr: "not-a-date",
			wantErr:    true,
		},
		{
			name:         "end_date before start_date is rejected",
			startDateStr: "2026-01-10",
			endDateStr:   "2026-01-01",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startTime, endTime, err := parseBookedAppointmentsDateRange(loc, tt.startDateStr, tt.endDateStr)

			if tt.wantErr {
				if !errors.Is(err, domain.ErrInvalidInput) {
					t.Fatalf("expected domain.ErrInvalidInput, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.startDateStr == "" && startTime != nil {
				t.Fatalf("expected nil startTime when start_date is empty, got %v", *startTime)
			}
			if tt.endDateStr == "" && endTime != nil {
				t.Fatalf("expected nil endTime when end_date is empty, got %v", *endTime)
			}
			if tt.startDateStr != "" && (startTime == nil || startTime.Hour() != 0) {
				t.Fatalf("expected startTime to be the start of day, got %v", startTime)
			}
			if tt.endDateStr != "" {
				if endTime == nil {
					t.Fatalf("expected non-nil endTime for end_date %q", tt.endDateStr)
				}
				nextDay := endTime.Add(time.Nanosecond)
				if nextDay.Hour() != 0 || nextDay.Minute() != 0 {
					t.Fatalf("expected endTime to be the last instant of the day, got %v", *endTime)
				}
			}
		})
	}
}

func TestListMyAppointments_ReturnsAppointmentsFromRepository(t *testing.T) {
	const userID = "client-1"

	repo := &fakeAppointmentRepository{
		listAppointmentsByClientFunc: func(ctx context.Context, clientID string) ([]*domain.AppointmentWithStore, error) {
			if clientID != userID {
				t.Fatalf("expected clientID %q, got %q", userID, clientID)
			}
			return []*domain.AppointmentWithStore{
				{Appointment: domain.Appointment{ID: "1", UserID: userID, Status: domain.AppointmentStatusConfirmed}, StoreName: "Store A"},
			}, nil
		},
	}
	uc := NewAppointmentUseCase(repo, nil, nil)

	appointments, err := uc.ListMyAppointments(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(appointments) != 1 || appointments[0].StoreName != "Store A" {
		t.Fatalf("unexpected result: %+v", appointments)
	}
}

func TestListMyAppointments_WrapsRepositoryError(t *testing.T) {
	repoErr := errors.New("db unavailable")
	repo := &fakeAppointmentRepository{
		listAppointmentsByClientFunc: func(ctx context.Context, clientID string) ([]*domain.AppointmentWithStore, error) {
			return nil, repoErr
		},
	}
	uc := NewAppointmentUseCase(repo, nil, nil)

	_, err := uc.ListMyAppointments(context.Background(), "client-1")
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected error to wrap %v, got %v", repoErr, err)
	}
}

func TestCancel(t *testing.T) {
	const (
		appointmentID = "appointment-1"
		clientID      = "client-1"
		ownerID       = "owner-1"
		strangerID    = "stranger-1"
		storeID       = "store-1"
	)

	tests := []struct {
		name            string
		appointment     *domain.Appointment
		requesterUserID string
		wantErr         error
	}{
		{
			name:            "the client who booked it can cancel a pending appointment",
			appointment:     &domain.Appointment{ID: appointmentID, StoreID: storeID, UserID: clientID, Status: domain.AppointmentStatusPending},
			requesterUserID: clientID,
		},
		{
			name:            "the store owner can cancel a confirmed appointment",
			appointment:     &domain.Appointment{ID: appointmentID, StoreID: storeID, UserID: clientID, Status: domain.AppointmentStatusConfirmed},
			requesterUserID: ownerID,
		},
		{
			name:            "a stranger cannot cancel it",
			appointment:     &domain.Appointment{ID: appointmentID, StoreID: storeID, UserID: clientID, Status: domain.AppointmentStatusPending},
			requesterUserID: strangerID,
			wantErr:         domain.ErrForbidden,
		},
		{
			name:            "an already cancelled appointment cannot be cancelled again",
			appointment:     &domain.Appointment{ID: appointmentID, StoreID: storeID, UserID: clientID, Status: domain.AppointmentStatusCancelled},
			requesterUserID: clientID,
			wantErr:         domain.ErrInvalidInput,
		},
		{
			name:            "a completed appointment cannot be cancelled",
			appointment:     &domain.Appointment{ID: appointmentID, StoreID: storeID, UserID: clientID, Status: domain.AppointmentStatusCompleted},
			requesterUserID: ownerID,
			wantErr:         domain.ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeAppointmentRepository{
				getAppointmentByIDFunc: func(ctx context.Context, id string) (*domain.Appointment, error) {
					return tt.appointment, nil
				},
				updateAppointmentStatusFunc: func(ctx context.Context, id, status string) (*domain.Appointment, error) {
					if status != domain.AppointmentStatusCancelled {
						t.Fatalf("expected status %q, got %q", domain.AppointmentStatusCancelled, status)
					}
					return &domain.Appointment{ID: id, StoreID: tt.appointment.StoreID, UserID: tt.appointment.UserID, Status: status}, nil
				},
			}
			storeRepo := &fakeAppointmentStoreRepository{
				getStoreByIDFunc: func(ctx context.Context, storeID string) (*domain.Store, error) {
					return &domain.Store{ID: storeID, UserID: ownerID}, nil
				},
			}
			uc := NewAppointmentUseCase(repo, nil, storeRepo)

			cancelled, err := uc.Cancel(context.Background(), appointmentID, tt.requesterUserID)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cancelled.Status != domain.AppointmentStatusCancelled {
				t.Fatalf("expected status %q, got %q", domain.AppointmentStatusCancelled, cancelled.Status)
			}
		})
	}
}

func TestExpireOverdueAppointments_WrapsRepositoryError(t *testing.T) {
	repoErr := errors.New("db unavailable")
	repo := &fakeAppointmentRepository{
		expireOverdueAppointmentsFunc: func(ctx context.Context, before time.Time) ([]*domain.Appointment, error) {
			return nil, repoErr
		},
	}
	uc := NewAppointmentUseCase(repo, nil, nil)

	_, err := uc.ExpireOverdueAppointments(context.Background())
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected error to wrap %v, got %v", repoErr, err)
	}
}
