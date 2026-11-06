package usecase

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/KevinManzano5/misto-backend/internal/domain"
)

// discardLogger is a *slog.Logger that writes nowhere, for tests that don't care about log output.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// fakeAppointmentRepository is a minimal AppointmentRepository test double; only the methods
// exercised by a given test need a non-nil func field.
type fakeAppointmentRepository struct {
	createAppointmentFunc                    func(ctx context.Context, appointment *domain.Appointment) error
	listAppointmentsByStoreDateFunc          func(ctx context.Context, storeID string, startTime, endTime time.Time) ([]*domain.Appointment, error)
	expireOverdueAppointmentsFunc            func(ctx context.Context, before time.Time) ([]*domain.Appointment, error)
	getAppointmentByIDFunc                   func(ctx context.Context, id string) (*domain.Appointment, error)
	getActiveAppointmentByStoreAndClientFunc func(ctx context.Context, storeID, clientID string) (*domain.Appointment, error)
	updateAppointmentStatusFunc              func(ctx context.Context, id, status, cancellationReason string) (*domain.Appointment, error)
	updatePaymentStatusFunc                  func(ctx context.Context, id, paymentStatus string) (*domain.Appointment, error)
	listAppointmentsByClientFunc             func(ctx context.Context, clientID string) ([]*domain.AppointmentWithStore, error)
}

func (f *fakeAppointmentRepository) CreateAppointment(ctx context.Context, appointment *domain.Appointment) error {
	if f.createAppointmentFunc != nil {
		return f.createAppointmentFunc(ctx, appointment)
	}
	panic("not implemented")
}

func (f *fakeAppointmentRepository) ListAppointmentsByStoreDate(ctx context.Context, storeID string, startTime, endTime time.Time) ([]*domain.Appointment, error) {
	if f.listAppointmentsByStoreDateFunc != nil {
		return f.listAppointmentsByStoreDateFunc(ctx, storeID, startTime, endTime)
	}
	panic("not implemented")
}

func (f *fakeAppointmentRepository) ListAllAppointmentsByStoreWithClient(ctx context.Context, storeID string, startTime, endTime *time.Time) ([]*domain.AppointmentWithClient, error) {
	panic("not implemented")
}

func (f *fakeAppointmentRepository) GetAppointmentByID(ctx context.Context, id string) (*domain.Appointment, error) {
	return f.getAppointmentByIDFunc(ctx, id)
}

func (f *fakeAppointmentRepository) GetActiveAppointmentByStoreAndClient(ctx context.Context, storeID, clientID string) (*domain.Appointment, error) {
	return f.getActiveAppointmentByStoreAndClientFunc(ctx, storeID, clientID)
}

func (f *fakeAppointmentRepository) UpdateAppointmentStatus(ctx context.Context, id, status, cancellationReason string) (*domain.Appointment, error) {
	return f.updateAppointmentStatusFunc(ctx, id, status, cancellationReason)
}

func (f *fakeAppointmentRepository) UpdatePaymentStatus(ctx context.Context, id, paymentStatus string) (*domain.Appointment, error) {
	return f.updatePaymentStatusFunc(ctx, id, paymentStatus)
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

// fakeStoreScheduleRepository is a minimal StoreScheduleRepository test double.
type fakeStoreScheduleRepository struct {
	getStoreScheduleByDayFunc func(ctx context.Context, storeID string, dayOfWeek int32) (*domain.StoreSchedule, error)
}

func (f *fakeStoreScheduleRepository) GetStoreScheduleByDay(ctx context.Context, storeID string, dayOfWeek int32) (*domain.StoreSchedule, error) {
	return f.getStoreScheduleByDayFunc(ctx, storeID, dayOfWeek)
}

// fakeAppointmentStoreServiceRepository is a minimal AppointmentStoreServiceRepository test double.
type fakeAppointmentStoreServiceRepository struct {
	getStoreServiceByIDFunc func(ctx context.Context, id string) (*domain.StoreService, error)
}

func (f *fakeAppointmentStoreServiceRepository) GetStoreServiceByID(ctx context.Context, id string) (*domain.StoreService, error) {
	return f.getStoreServiceByIDFunc(ctx, id)
}

// fakePaymentProcessor is a minimal PaymentProcessor test double; chargeFunc defaults to success.
type fakePaymentProcessor struct {
	chargeFunc func(ctx context.Context, appointment *domain.Appointment) error
}

func (f *fakePaymentProcessor) Charge(ctx context.Context, appointment *domain.Appointment) error {
	if f.chargeFunc != nil {
		return f.chargeFunc(ctx, appointment)
	}
	return nil
}

// fakePaymentTransactionRepository is a minimal PaymentTransactionRepository test double that
// records every CreatePaymentTransaction call, so tests can assert on the resulting history.
type fakePaymentTransactionRepository struct {
	created   []*domain.PaymentTransaction
	createErr error
}

func (f *fakePaymentTransactionRepository) CreatePaymentTransaction(ctx context.Context, transaction *domain.PaymentTransaction) error {
	if f.createErr != nil {
		return f.createErr
	}
	transaction.ID = "fake-transaction-id"
	f.created = append(f.created, transaction)
	return nil
}

func (f *fakePaymentTransactionRepository) ListPaymentTransactionsByAppointment(ctx context.Context, appointmentID string) ([]*domain.PaymentTransaction, error) {
	return f.created, nil
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
	uc := NewAppointmentUseCase(repo, nil, nil, nil, &fakePaymentProcessor{}, &fakePaymentTransactionRepository{}, discardLogger())

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
	uc := NewAppointmentUseCase(repo, nil, nil, nil, &fakePaymentProcessor{}, &fakePaymentTransactionRepository{}, discardLogger())

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
	uc := NewAppointmentUseCase(repo, nil, nil, nil, &fakePaymentProcessor{}, &fakePaymentTransactionRepository{}, discardLogger())

	_, err := uc.ListMyAppointments(context.Background(), "client-1")
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected error to wrap %v, got %v", repoErr, err)
	}
}

func TestGetActiveAppointmentStatus(t *testing.T) {
	repoErr := errors.New("db unavailable")
	active := &domain.Appointment{ID: "appt-1", StoreID: "store-1", UserID: "client-1", Status: domain.AppointmentStatusConfirmed}

	tests := []struct {
		name     string
		repoFunc func(ctx context.Context, storeID, clientID string) (*domain.Appointment, error)
		want     *domain.Appointment
		wantErr  error
	}{
		{
			name:     "an active appointment is returned",
			repoFunc: func(ctx context.Context, storeID, clientID string) (*domain.Appointment, error) { return active, nil },
			want:     active,
		},
		{
			name:     "no active appointment returns nil, nil, not an error",
			repoFunc: func(ctx context.Context, storeID, clientID string) (*domain.Appointment, error) { return nil, nil },
			want:     nil,
		},
		{
			name:     "a repository error is wrapped and propagated",
			repoFunc: func(ctx context.Context, storeID, clientID string) (*domain.Appointment, error) { return nil, repoErr },
			wantErr:  repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeAppointmentRepository{getActiveAppointmentByStoreAndClientFunc: tt.repoFunc}
			uc := NewAppointmentUseCase(repo, nil, nil, nil, &fakePaymentProcessor{}, &fakePaymentTransactionRepository{}, discardLogger())

			got, err := uc.GetActiveAppointmentStatus(context.Background(), "store-1", "client-1")

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error to wrap %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %+v, got %+v", tt.want, got)
			}
		})
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
				updateAppointmentStatusFunc: func(ctx context.Context, id, status, cancellationReason string) (*domain.Appointment, error) {
					if status != domain.AppointmentStatusCancelled {
						t.Fatalf("expected status %q, got %q", domain.AppointmentStatusCancelled, status)
					}
					return &domain.Appointment{ID: id, StoreID: tt.appointment.StoreID, UserID: tt.appointment.UserID, Status: status, CancellationReason: cancellationReason}, nil
				},
			}
			storeRepo := &fakeAppointmentStoreRepository{
				getStoreByIDFunc: func(ctx context.Context, storeID string) (*domain.Store, error) {
					return &domain.Store{ID: storeID, UserID: ownerID}, nil
				},
			}
			uc := NewAppointmentUseCase(repo, nil, storeRepo, nil, &fakePaymentProcessor{}, &fakePaymentTransactionRepository{}, discardLogger())

			cancelled, err := uc.Cancel(context.Background(), appointmentID, tt.requesterUserID, "no longer needed")

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

func TestCancel_RejectsEmptyReason(t *testing.T) {
	uc := NewAppointmentUseCase(&fakeAppointmentRepository{}, nil, &fakeAppointmentStoreRepository{}, nil, &fakePaymentProcessor{}, &fakePaymentTransactionRepository{}, discardLogger())

	if _, err := uc.Cancel(context.Background(), "appointment-1", "client-1", "   "); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected %v, got %v", domain.ErrInvalidInput, err)
	}
}

func TestExpireOverdueAppointments_WrapsRepositoryError(t *testing.T) {
	repoErr := errors.New("db unavailable")
	repo := &fakeAppointmentRepository{
		expireOverdueAppointmentsFunc: func(ctx context.Context, before time.Time) ([]*domain.Appointment, error) {
			return nil, repoErr
		},
	}
	uc := NewAppointmentUseCase(repo, nil, nil, nil, &fakePaymentProcessor{}, &fakePaymentTransactionRepository{}, discardLogger())

	_, err := uc.ExpireOverdueAppointments(context.Background())
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected error to wrap %v, got %v", repoErr, err)
	}
}

// newCreateTestStores builds the store/schedule/service repo fakes shared by the Create payment
// tests: a store whose schedule is open all day, every day, with a single active service
// ("service-1") belonging to it, so only payment behavior is under test.
func newCreateTestStores(autoConfirm bool) (*fakeAppointmentStoreRepository, *fakeStoreScheduleRepository, *fakeAppointmentStoreServiceRepository) {
	storeRepo := &fakeAppointmentStoreRepository{
		getStoreByIDFunc: func(ctx context.Context, storeID string) (*domain.Store, error) {
			return &domain.Store{
				ID:                       storeID,
				UserID:                   "owner-1",
				Timezone:                 "UTC",
				AppointmentDuration:      30,
				AllowSameDayAppointments: true,
				AutoConfirmAppointments:  autoConfirm,
			}, nil
		},
	}
	scheduleRepo := &fakeStoreScheduleRepository{
		getStoreScheduleByDayFunc: func(ctx context.Context, storeID string, dayOfWeek int32) (*domain.StoreSchedule, error) {
			return &domain.StoreSchedule{StoreID: storeID, DayOfWeek: dayOfWeek, OpenTime: "00:00", CloseTime: "23:59"}, nil
		},
	}
	serviceRepo := &fakeAppointmentStoreServiceRepository{
		getStoreServiceByIDFunc: func(ctx context.Context, id string) (*domain.StoreService, error) {
			return &domain.StoreService{ID: id, StoreID: "store-1", DurationMinutes: 30, IsActive: true}, nil
		},
	}
	return storeRepo, scheduleRepo, serviceRepo
}

func TestCreate_RejectsInvalidPaymentMethod(t *testing.T) {
	uc := NewAppointmentUseCase(&fakeAppointmentRepository{}, nil, nil, nil, &fakePaymentProcessor{}, &fakePaymentTransactionRepository{}, discardLogger())

	_, err := uc.Create(context.Background(), "store-1", "service-1", "client-1", "bitcoin", time.Now().Add(time.Hour))
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected %v, got %v", domain.ErrInvalidInput, err)
	}
}

func TestCreate_RejectsServiceNotBelongingToStore(t *testing.T) {
	storeRepo, scheduleRepo, _ := newCreateTestStores(true)
	serviceRepo := &fakeAppointmentStoreServiceRepository{
		getStoreServiceByIDFunc: func(ctx context.Context, id string) (*domain.StoreService, error) {
			return &domain.StoreService{ID: id, StoreID: "other-store", DurationMinutes: 30, IsActive: true}, nil
		},
	}
	uc := NewAppointmentUseCase(&fakeAppointmentRepository{}, scheduleRepo, storeRepo, serviceRepo, &fakePaymentProcessor{}, &fakePaymentTransactionRepository{}, discardLogger())

	_, err := uc.Create(context.Background(), "store-1", "service-1", "client-1", domain.PaymentMethodCashOnSite, time.Now().Add(time.Hour))
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected %v, got %v", domain.ErrInvalidInput, err)
	}
}

func TestCreate_RejectsInactiveService(t *testing.T) {
	storeRepo, scheduleRepo, _ := newCreateTestStores(true)
	serviceRepo := &fakeAppointmentStoreServiceRepository{
		getStoreServiceByIDFunc: func(ctx context.Context, id string) (*domain.StoreService, error) {
			return &domain.StoreService{ID: id, StoreID: "store-1", DurationMinutes: 30, IsActive: false}, nil
		},
	}
	uc := NewAppointmentUseCase(&fakeAppointmentRepository{}, scheduleRepo, storeRepo, serviceRepo, &fakePaymentProcessor{}, &fakePaymentTransactionRepository{}, discardLogger())

	_, err := uc.Create(context.Background(), "store-1", "service-1", "client-1", domain.PaymentMethodCashOnSite, time.Now().Add(time.Hour))
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected %v, got %v", domain.ErrInvalidInput, err)
	}
}

func TestCreate_EndTimeReflectsServiceDuration(t *testing.T) {
	storeRepo, scheduleRepo, _ := newCreateTestStores(true)
	serviceRepo := &fakeAppointmentStoreServiceRepository{
		getStoreServiceByIDFunc: func(ctx context.Context, id string) (*domain.StoreService, error) {
			return &domain.StoreService{ID: id, StoreID: "store-1", DurationMinutes: 45, IsActive: true}, nil
		},
	}
	repo := &fakeAppointmentRepository{
		createAppointmentFunc: func(ctx context.Context, appointment *domain.Appointment) error { return nil },
	}
	uc := NewAppointmentUseCase(repo, scheduleRepo, storeRepo, serviceRepo, &fakePaymentProcessor{}, &fakePaymentTransactionRepository{}, discardLogger())

	startTime := time.Now().Add(time.Hour)
	appointment, err := uc.Create(context.Background(), "store-1", "service-1", "client-1", domain.PaymentMethodCashOnSite, startTime)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := appointment.EndTime.Sub(appointment.StartTime); got != 45*time.Minute {
		t.Fatalf("expected appointment duration to reflect the service's 45-minute duration (not the store's 30-minute default), got %v", got)
	}
	if appointment.ServiceID != "service-1" {
		t.Fatalf("expected ServiceID %q, got %q", "service-1", appointment.ServiceID)
	}
}

func TestCreate_CashOnSite_NeverCharges(t *testing.T) {
	storeRepo, scheduleRepo, serviceRepo := newCreateTestStores(true)
	repo := &fakeAppointmentRepository{
		createAppointmentFunc: func(ctx context.Context, appointment *domain.Appointment) error { return nil },
	}
	charged := false
	processor := &fakePaymentProcessor{chargeFunc: func(ctx context.Context, appointment *domain.Appointment) error {
		charged = true
		return nil
	}}
	transactionRepo := &fakePaymentTransactionRepository{}
	uc := NewAppointmentUseCase(repo, scheduleRepo, storeRepo, serviceRepo, processor, transactionRepo, discardLogger())

	appointment, err := uc.Create(context.Background(), "store-1", "service-1", "client-1", domain.PaymentMethodCashOnSite, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if charged {
		t.Fatal("expected cash_on_site to never call the payment processor")
	}
	if appointment.PaymentStatus != domain.PaymentStatusPending {
		t.Fatalf("expected payment status %q, got %q", domain.PaymentStatusPending, appointment.PaymentStatus)
	}
	if appointment.Status != domain.AppointmentStatusConfirmed {
		t.Fatalf("expected status %q, got %q", domain.AppointmentStatusConfirmed, appointment.Status)
	}
	if len(transactionRepo.created) != 1 || transactionRepo.created[0].Status != domain.PaymentStatusPending {
		t.Fatalf("expected exactly one pending payment transaction to be recorded for cash_on_site, got %+v", transactionRepo.created)
	}
}

func TestCreate_CardAdvance_AutoConfirm_ChargeSucceeds(t *testing.T) {
	storeRepo, scheduleRepo, serviceRepo := newCreateTestStores(true)
	repo := &fakeAppointmentRepository{
		createAppointmentFunc: func(ctx context.Context, appointment *domain.Appointment) error { return nil },
	}
	transactionRepo := &fakePaymentTransactionRepository{}
	uc := NewAppointmentUseCase(repo, scheduleRepo, storeRepo, serviceRepo, &fakePaymentProcessor{}, transactionRepo, discardLogger())

	appointment, err := uc.Create(context.Background(), "store-1", "service-1", "client-1", domain.PaymentMethodCardAdvance, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if appointment.Status != domain.AppointmentStatusConfirmed {
		t.Fatalf("expected status %q, got %q", domain.AppointmentStatusConfirmed, appointment.Status)
	}
	if appointment.PaymentStatus != domain.PaymentStatusPaid {
		t.Fatalf("expected payment status %q, got %q", domain.PaymentStatusPaid, appointment.PaymentStatus)
	}
	if len(transactionRepo.created) != 1 || transactionRepo.created[0].Status != domain.PaymentStatusPaid {
		t.Fatalf("expected exactly one paid payment transaction to be recorded, got %+v", transactionRepo.created)
	}
}

func TestCreate_CardAdvance_AutoConfirm_ChargeFails(t *testing.T) {
	storeRepo, scheduleRepo, serviceRepo := newCreateTestStores(true)
	repo := &fakeAppointmentRepository{
		createAppointmentFunc: func(ctx context.Context, appointment *domain.Appointment) error { return nil },
	}
	processor := &fakePaymentProcessor{chargeFunc: func(ctx context.Context, appointment *domain.Appointment) error {
		return errors.New("card declined")
	}}
	transactionRepo := &fakePaymentTransactionRepository{}
	uc := NewAppointmentUseCase(repo, scheduleRepo, storeRepo, serviceRepo, processor, transactionRepo, discardLogger())

	appointment, err := uc.Create(context.Background(), "store-1", "service-1", "client-1", domain.PaymentMethodCardAdvance, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if appointment.Status != domain.AppointmentStatusPending {
		t.Fatalf("expected status %q when the charge fails, got %q", domain.AppointmentStatusPending, appointment.Status)
	}
	if appointment.PaymentStatus != domain.PaymentStatusFailed {
		t.Fatalf("expected payment status %q, got %q", domain.PaymentStatusFailed, appointment.PaymentStatus)
	}
	if len(transactionRepo.created) != 1 || transactionRepo.created[0].Status != domain.PaymentStatusFailed || transactionRepo.created[0].FailureReason == "" {
		t.Fatalf("expected exactly one failed payment transaction with a failure reason, got %+v", transactionRepo.created)
	}
}

func TestCreate_CardAdvance_AutoConfirm_TransactionWriteFailureDoesNotBlockCreate(t *testing.T) {
	storeRepo, scheduleRepo, serviceRepo := newCreateTestStores(true)
	repo := &fakeAppointmentRepository{
		createAppointmentFunc: func(ctx context.Context, appointment *domain.Appointment) error { return nil },
	}
	transactionRepo := &fakePaymentTransactionRepository{createErr: errors.New("audit db unavailable")}
	uc := NewAppointmentUseCase(repo, scheduleRepo, storeRepo, serviceRepo, &fakePaymentProcessor{}, transactionRepo, discardLogger())

	appointment, err := uc.Create(context.Background(), "store-1", "service-1", "client-1", domain.PaymentMethodCardAdvance, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("expected the appointment to be created despite the audit write failing, got error: %v", err)
	}
	if appointment.PaymentStatus != domain.PaymentStatusPaid {
		t.Fatalf("expected payment status %q, got %q", domain.PaymentStatusPaid, appointment.PaymentStatus)
	}
}

func TestCreate_CardAdvance_ManualConfirm_DoesNotChargeYet(t *testing.T) {
	storeRepo, scheduleRepo, serviceRepo := newCreateTestStores(false)
	repo := &fakeAppointmentRepository{
		createAppointmentFunc: func(ctx context.Context, appointment *domain.Appointment) error { return nil },
	}
	charged := false
	processor := &fakePaymentProcessor{chargeFunc: func(ctx context.Context, appointment *domain.Appointment) error {
		charged = true
		return nil
	}}
	transactionRepo := &fakePaymentTransactionRepository{}
	uc := NewAppointmentUseCase(repo, scheduleRepo, storeRepo, serviceRepo, processor, transactionRepo, discardLogger())

	appointment, err := uc.Create(context.Background(), "store-1", "service-1", "client-1", domain.PaymentMethodCardAdvance, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if charged {
		t.Fatal("expected the charge to be deferred until the store confirms the appointment")
	}
	if appointment.Status != domain.AppointmentStatusPending {
		t.Fatalf("expected status %q, got %q", domain.AppointmentStatusPending, appointment.Status)
	}
	if appointment.PaymentStatus != domain.PaymentStatusPending {
		t.Fatalf("expected payment status %q, got %q", domain.PaymentStatusPending, appointment.PaymentStatus)
	}
	if len(transactionRepo.created) != 1 || transactionRepo.created[0].Status != domain.PaymentStatusPending {
		t.Fatalf("expected exactly one pending payment transaction to be recorded while the charge is deferred, got %+v", transactionRepo.created)
	}
}

func TestGetAvailableSlots_RejectsServiceNotBelongingToStore(t *testing.T) {
	storeRepo, scheduleRepo, _ := newCreateTestStores(true)
	serviceRepo := &fakeAppointmentStoreServiceRepository{
		getStoreServiceByIDFunc: func(ctx context.Context, id string) (*domain.StoreService, error) {
			return &domain.StoreService{ID: id, StoreID: "other-store", DurationMinutes: 30, IsActive: true}, nil
		},
	}
	uc := NewAppointmentUseCase(&fakeAppointmentRepository{}, scheduleRepo, storeRepo, serviceRepo, &fakePaymentProcessor{}, &fakePaymentTransactionRepository{}, discardLogger())

	dateStr := time.Now().Add(48 * time.Hour).Format("2006-01-02")
	_, err := uc.GetAvailableSlots(context.Background(), "store-1", "service-1", dateStr)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected %v, got %v", domain.ErrInvalidInput, err)
	}
}

func TestGetAvailableSlots_RejectsInactiveService(t *testing.T) {
	storeRepo, scheduleRepo, _ := newCreateTestStores(true)
	serviceRepo := &fakeAppointmentStoreServiceRepository{
		getStoreServiceByIDFunc: func(ctx context.Context, id string) (*domain.StoreService, error) {
			return &domain.StoreService{ID: id, StoreID: "store-1", DurationMinutes: 30, IsActive: false}, nil
		},
	}
	uc := NewAppointmentUseCase(&fakeAppointmentRepository{}, scheduleRepo, storeRepo, serviceRepo, &fakePaymentProcessor{}, &fakePaymentTransactionRepository{}, discardLogger())

	dateStr := time.Now().Add(48 * time.Hour).Format("2006-01-02")
	_, err := uc.GetAvailableSlots(context.Background(), "store-1", "service-1", dateStr)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected %v, got %v", domain.ErrInvalidInput, err)
	}
}

func TestGetAvailableSlots_SpacingReflectsServiceDuration(t *testing.T) {
	storeRepo, scheduleRepo, _ := newCreateTestStores(true)
	serviceRepo := &fakeAppointmentStoreServiceRepository{
		getStoreServiceByIDFunc: func(ctx context.Context, id string) (*domain.StoreService, error) {
			return &domain.StoreService{ID: id, StoreID: "store-1", DurationMinutes: 60, IsActive: true}, nil
		},
	}
	repo := &fakeAppointmentRepository{
		listAppointmentsByStoreDateFunc: func(ctx context.Context, storeID string, startTime, endTime time.Time) ([]*domain.Appointment, error) {
			return nil, nil
		},
	}
	uc := NewAppointmentUseCase(repo, scheduleRepo, storeRepo, serviceRepo, &fakePaymentProcessor{}, &fakePaymentTransactionRepository{}, discardLogger())

	// The schedule fake is open 00:00-23:59, so a 60-minute service produces 23 one-hour slots
	// (00:00 through 22:00 inclusive; a slot starting at 23:00 would end at 00:00 the next day,
	// past the 23:59 close time, so it doesn't fit).
	dateStr := time.Now().Add(48 * time.Hour).Format("2006-01-02")
	slots, err := uc.GetAvailableSlots(context.Background(), "store-1", "service-1", dateStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(slots) != 23 {
		t.Fatalf("expected 23 one-hour slots in a 00:00-23:59 window, got %d: %v", len(slots), slots)
	}
	if slots[1] != "01:00" {
		t.Fatalf("expected slots spaced by the service's 60-minute duration (second slot at 01:00), got %v", slots)
	}
}

func TestUpdateStatus_ConfirmingCardAdvance_ChargesAndConfirms(t *testing.T) {
	const appointmentID, ownerID, storeID = "appointment-1", "owner-1", "store-1"

	pending := &domain.Appointment{ID: appointmentID, StoreID: storeID, Status: domain.AppointmentStatusPending, PaymentMethod: domain.PaymentMethodCardAdvance, PaymentStatus: domain.PaymentStatusPending}

	var recordedPaymentStatus string
	repo := &fakeAppointmentRepository{
		getAppointmentByIDFunc: func(ctx context.Context, id string) (*domain.Appointment, error) { return pending, nil },
		updatePaymentStatusFunc: func(ctx context.Context, id, paymentStatus string) (*domain.Appointment, error) {
			recordedPaymentStatus = paymentStatus
			return pending, nil
		},
		updateAppointmentStatusFunc: func(ctx context.Context, id, status, cancellationReason string) (*domain.Appointment, error) {
			return &domain.Appointment{ID: id, StoreID: storeID, Status: status, PaymentMethod: domain.PaymentMethodCardAdvance, PaymentStatus: recordedPaymentStatus}, nil
		},
	}
	storeRepo := &fakeAppointmentStoreRepository{
		getStoreByIDFunc: func(ctx context.Context, storeID string) (*domain.Store, error) {
			return &domain.Store{ID: storeID, UserID: ownerID}, nil
		},
	}
	transactionRepo := &fakePaymentTransactionRepository{}
	uc := NewAppointmentUseCase(repo, nil, storeRepo, nil, &fakePaymentProcessor{}, transactionRepo, discardLogger())

	updated, err := uc.UpdateStatus(context.Background(), appointmentID, ownerID, domain.AppointmentStatusConfirmed, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Status != domain.AppointmentStatusConfirmed {
		t.Fatalf("expected status %q, got %q", domain.AppointmentStatusConfirmed, updated.Status)
	}
	if recordedPaymentStatus != domain.PaymentStatusPaid {
		t.Fatalf("expected payment status %q to be persisted, got %q", domain.PaymentStatusPaid, recordedPaymentStatus)
	}
	if len(transactionRepo.created) != 1 || transactionRepo.created[0].Status != domain.PaymentStatusPaid {
		t.Fatalf("expected exactly one paid payment transaction to be recorded, got %+v", transactionRepo.created)
	}
}

func TestUpdateStatus_ConfirmingCardAdvance_ChargeFailsKeepsItPending(t *testing.T) {
	const appointmentID, ownerID, storeID = "appointment-1", "owner-1", "store-1"

	pending := &domain.Appointment{ID: appointmentID, StoreID: storeID, Status: domain.AppointmentStatusPending, PaymentMethod: domain.PaymentMethodCardAdvance, PaymentStatus: domain.PaymentStatusPending}

	var recordedPaymentStatus string
	repo := &fakeAppointmentRepository{
		getAppointmentByIDFunc: func(ctx context.Context, id string) (*domain.Appointment, error) { return pending, nil },
		updatePaymentStatusFunc: func(ctx context.Context, id, paymentStatus string) (*domain.Appointment, error) {
			recordedPaymentStatus = paymentStatus
			return pending, nil
		},
		updateAppointmentStatusFunc: func(ctx context.Context, id, status, cancellationReason string) (*domain.Appointment, error) {
			t.Fatal("status should not be confirmed when the charge fails")
			return nil, nil
		},
	}
	storeRepo := &fakeAppointmentStoreRepository{
		getStoreByIDFunc: func(ctx context.Context, storeID string) (*domain.Store, error) {
			return &domain.Store{ID: storeID, UserID: ownerID}, nil
		},
	}
	processor := &fakePaymentProcessor{chargeFunc: func(ctx context.Context, appointment *domain.Appointment) error {
		return errors.New("card declined")
	}}
	transactionRepo := &fakePaymentTransactionRepository{}
	uc := NewAppointmentUseCase(repo, nil, storeRepo, nil, processor, transactionRepo, discardLogger())

	_, err := uc.UpdateStatus(context.Background(), appointmentID, ownerID, domain.AppointmentStatusConfirmed, "")
	if !errors.Is(err, domain.ErrPaymentFailed) {
		t.Fatalf("expected %v, got %v", domain.ErrPaymentFailed, err)
	}
	if recordedPaymentStatus != domain.PaymentStatusFailed {
		t.Fatalf("expected payment status %q to be persisted, got %q", domain.PaymentStatusFailed, recordedPaymentStatus)
	}
	if len(transactionRepo.created) != 1 || transactionRepo.created[0].Status != domain.PaymentStatusFailed || transactionRepo.created[0].FailureReason == "" {
		t.Fatalf("expected exactly one failed payment transaction with a failure reason, got %+v", transactionRepo.created)
	}
}

func TestConfirmCashPayment(t *testing.T) {
	const appointmentID, ownerID, strangerID, storeID = "appointment-1", "owner-1", "stranger-1", "store-1"

	tests := []struct {
		name            string
		appointment     *domain.Appointment
		requesterUserID string
		wantErr         error
	}{
		{
			name:            "the store owner can confirm a pending cash payment",
			appointment:     &domain.Appointment{ID: appointmentID, StoreID: storeID, PaymentMethod: domain.PaymentMethodCashOnSite, PaymentStatus: domain.PaymentStatusPending},
			requesterUserID: ownerID,
		},
		{
			name:            "a stranger cannot confirm it",
			appointment:     &domain.Appointment{ID: appointmentID, StoreID: storeID, PaymentMethod: domain.PaymentMethodCashOnSite, PaymentStatus: domain.PaymentStatusPending},
			requesterUserID: strangerID,
			wantErr:         domain.ErrForbidden,
		},
		{
			name:            "a card_advance appointment cannot be confirmed this way",
			appointment:     &domain.Appointment{ID: appointmentID, StoreID: storeID, PaymentMethod: domain.PaymentMethodCardAdvance, PaymentStatus: domain.PaymentStatusPending},
			requesterUserID: ownerID,
			wantErr:         domain.ErrInvalidInput,
		},
		{
			name:            "an already paid cash payment cannot be confirmed again",
			appointment:     &domain.Appointment{ID: appointmentID, StoreID: storeID, PaymentMethod: domain.PaymentMethodCashOnSite, PaymentStatus: domain.PaymentStatusPaid},
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
				updatePaymentStatusFunc: func(ctx context.Context, id, paymentStatus string) (*domain.Appointment, error) {
					return &domain.Appointment{ID: id, StoreID: tt.appointment.StoreID, PaymentMethod: tt.appointment.PaymentMethod, PaymentStatus: paymentStatus}, nil
				},
			}
			storeRepo := &fakeAppointmentStoreRepository{
				getStoreByIDFunc: func(ctx context.Context, storeID string) (*domain.Store, error) {
					return &domain.Store{ID: storeID, UserID: ownerID}, nil
				},
			}
			transactionRepo := &fakePaymentTransactionRepository{}
			uc := NewAppointmentUseCase(repo, nil, storeRepo, nil, &fakePaymentProcessor{}, transactionRepo, discardLogger())

			updated, err := uc.ConfirmCashPayment(context.Background(), appointmentID, tt.requesterUserID)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if updated.PaymentStatus != domain.PaymentStatusPaid {
				t.Fatalf("expected payment status %q, got %q", domain.PaymentStatusPaid, updated.PaymentStatus)
			}
			if len(transactionRepo.created) != 1 || transactionRepo.created[0].Status != domain.PaymentStatusPaid {
				t.Fatalf("expected exactly one paid payment transaction to be recorded, got %+v", transactionRepo.created)
			}
		})
	}
}

func TestListPaymentTransactions_Authorization(t *testing.T) {
	const appointmentID, clientID, ownerID, strangerID, storeID = "appointment-1", "client-1", "owner-1", "stranger-1", "store-1"

	appointment := &domain.Appointment{ID: appointmentID, StoreID: storeID, UserID: clientID}
	transactions := []*domain.PaymentTransaction{{ID: "tx-1", AppointmentID: appointmentID, Status: domain.PaymentStatusPaid}}

	tests := []struct {
		name            string
		requesterUserID string
		wantErr         error
	}{
		{name: "the client who booked it can view its transactions", requesterUserID: clientID},
		{name: "the store owner can view its transactions", requesterUserID: ownerID},
		{name: "a stranger cannot view its transactions", requesterUserID: strangerID, wantErr: domain.ErrForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeAppointmentRepository{
				getAppointmentByIDFunc: func(ctx context.Context, id string) (*domain.Appointment, error) {
					return appointment, nil
				},
			}
			storeRepo := &fakeAppointmentStoreRepository{
				getStoreByIDFunc: func(ctx context.Context, storeID string) (*domain.Store, error) {
					return &domain.Store{ID: storeID, UserID: ownerID}, nil
				},
			}
			transactionRepo := &fakePaymentTransactionRepository{created: transactions}
			uc := NewAppointmentUseCase(repo, nil, storeRepo, nil, &fakePaymentProcessor{}, transactionRepo, discardLogger())

			got, err := uc.ListPaymentTransactions(context.Background(), appointmentID, tt.requesterUserID)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != 1 || got[0].ID != "tx-1" {
				t.Fatalf("unexpected result: %+v", got)
			}
		})
	}
}
