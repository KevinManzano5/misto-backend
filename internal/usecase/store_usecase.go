package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/KevinManzano5/misto-backend/internal/domain"
)

// defaultOpenTime and defaultCloseTime are the opening hours applied to every day of the week
// when a store is created, until the owner customizes its schedule.
const (
	defaultOpenTime  = "09:00"
	defaultCloseTime = "18:00"
)

// sunday is the domain.StoreSchedule.DayOfWeek value for Sunday, closed by default.
const sunday = 0

// defaultAppointmentDuration is applied to a store when it is created without one.
const defaultAppointmentDuration = 30

// defaultTimezone is applied to a store when it is created without one.
const defaultTimezone = "UTC"

// validateTimezone checks that name is a loadable IANA timezone (e.g. "America/Bogota").
func validateTimezone(name string) error {
	if _, err := time.LoadLocation(name); err != nil {
		return fmt.Errorf("%w: invalid timezone %q", domain.ErrInvalidInput, name)
	}

	return nil
}

// StoreRepository interface
type StoreRepository interface {
	CreateStore(ctx context.Context, store *domain.Store) error
	CreateStoreWithSchedules(ctx context.Context, store *domain.Store, schedules []*domain.StoreSchedule) error
	GetStoreByID(ctx context.Context, storeID string) (*domain.Store, error)
	GetStoreByUserID(ctx context.Context, userID string) (*domain.Store, error)
	UpdateStore(ctx context.Context, store *domain.Store) error
	DeleteStore(ctx context.Context, id string) error
	ListStores(ctx context.Context) ([]*domain.Store, error)
}

// ScheduleRepository interface
type ScheduleRepository interface {
	UpsertStoreSchedule(ctx context.Context, schedule *domain.StoreSchedule) error
	ListStoreSchedules(ctx context.Context, storeID string) ([]*domain.StoreSchedule, error)
}

// StoreUseCase struct
type StoreUseCase struct {
	repo         StoreRepository
	scheduleRepo ScheduleRepository
}

// NewStoreUseCase constructor
func NewStoreUseCase(repo StoreRepository, scheduleRepo ScheduleRepository) *StoreUseCase {
	return &StoreUseCase{
		repo:         repo,
		scheduleRepo: scheduleRepo,
	}
}

// Create method creates a store along with a default weekly schedule (open every day
// 09:00-18:00, except Sunday which is closed), which the owner can later customize.
func (usecase *StoreUseCase) Create(ctx context.Context, userID, name, description string, isActive bool, appointmentDuration int32, timezone string, autoConfirmAppointments bool) (*domain.Store, error) {
	if appointmentDuration <= 0 {
		appointmentDuration = defaultAppointmentDuration
	}

	if timezone == "" {
		timezone = defaultTimezone
	} else if err := validateTimezone(timezone); err != nil {
		return nil, err
	}

	store := domain.NewStore(userID, name, description, isActive, appointmentDuration, timezone, autoConfirmAppointments)
	schedules := defaultWeeklySchedules()

	if err := usecase.repo.CreateStoreWithSchedules(ctx, store, schedules); err != nil {
		return nil, fmt.Errorf("creating store: %w", err)
	}

	return store, nil
}

// defaultWeeklySchedules builds the default opening-hours schedule for all seven days of the
// week, applied to every newly created store.
func defaultWeeklySchedules() []*domain.StoreSchedule {
	schedules := make([]*domain.StoreSchedule, 0, 7)

	for dayOfWeek := int32(0); dayOfWeek <= 6; dayOfWeek++ {
		isClosed := dayOfWeek == sunday
		schedules = append(schedules, domain.NewStoreSchedule("", dayOfWeek, defaultOpenTime, defaultCloseTime, isClosed))
	}

	return schedules
}

// GetByID method
func (usecase *StoreUseCase) GetByID(ctx context.Context, storeID string) (*domain.Store, error) {
	store, err := usecase.repo.GetStoreByID(ctx, storeID)

	if err != nil {
		return nil, fmt.Errorf("getting store: %w", err)
	}

	return store, nil
}

// GetByUserID method
func (usecase *StoreUseCase) GetByUserID(ctx context.Context, userID string) (*domain.Store, error) {
	store, err := usecase.repo.GetStoreByUserID(ctx, userID)

	if err != nil {
		return nil, fmt.Errorf("getting store: %w", err)
	}

	return store, nil
}

// List method
func (usecase *StoreUseCase) List(ctx context.Context) ([]*domain.Store, error) {
	stores, err := usecase.repo.ListStores(ctx)

	if err != nil {
		return nil, fmt.Errorf("listing stores: %w", err)
	}

	return stores, nil
}

// Update method applies the supplied changes to a store and returns the persisted store.
func (usecase *StoreUseCase) Update(ctx context.Context, storeID string, name, description *string, isActive *bool, appointmentDuration *int32, timezone *string, autoConfirmAppointments *bool) (*domain.Store, error) {
	store, err := usecase.repo.GetStoreByID(ctx, storeID)
	if err != nil {
		return nil, fmt.Errorf("getting store to update: %w", err)
	}

	if name != nil {
		store.Name = *name
	}
	if description != nil {
		store.Description = *description
	}
	if isActive != nil {
		store.IsActive = *isActive
	}
	if appointmentDuration != nil {
		store.AppointmentDuration = *appointmentDuration
	}
	if timezone != nil {
		if err := validateTimezone(*timezone); err != nil {
			return nil, err
		}
		store.Timezone = *timezone
	}
	if autoConfirmAppointments != nil {
		store.AutoConfirmAppointments = *autoConfirmAppointments
	}

	if !store.IsValid() {
		return nil, domain.ErrInvalidInput
	}

	if err := usecase.repo.UpdateStore(ctx, store); err != nil {
		return nil, fmt.Errorf("updating store: %w", err)
	}

	updatedStore, err := usecase.repo.GetStoreByID(ctx, storeID)
	if err != nil {
		return nil, fmt.Errorf("getting updated store: %w", err)
	}

	return updatedStore, nil
}

// Delete method
func (usecase *StoreUseCase) Delete(ctx context.Context, storeID string) error {
	err := usecase.repo.DeleteStore(ctx, storeID)

	if err != nil {
		return fmt.Errorf("deleting store: %w", err)
	}

	return nil
}

// UpsertSchedule method creates or updates the schedule of a store for a specific day
func (usecase *StoreUseCase) UpsertSchedule(ctx context.Context, storeID string, dayOfWeek int32, openTime, closeTime string, isClosed bool) (*domain.StoreSchedule, error) {
	schedule := domain.NewStoreSchedule(storeID, dayOfWeek, openTime, closeTime, isClosed)

	if !schedule.IsValid() {
		return nil, domain.ErrInvalidInput
	}

	if err := usecase.scheduleRepo.UpsertStoreSchedule(ctx, schedule); err != nil {
		return nil, fmt.Errorf("upserting store schedule: %w", err)
	}

	return schedule, nil
}

// ListSchedules method retrieves all schedules for a store
func (usecase *StoreUseCase) ListSchedules(ctx context.Context, storeID string) ([]*domain.StoreSchedule, error) {
	schedules, err := usecase.scheduleRepo.ListStoreSchedules(ctx, storeID)
	if err != nil {
		return nil, fmt.Errorf("listing store schedules: %w", err)
	}

	return schedules, nil
}
