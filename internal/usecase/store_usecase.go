package usecase

import (
	"context"
	"fmt"

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
	GetStoreScheduleByDay(ctx context.Context, storeID string, dayOfWeek int32) (*domain.StoreSchedule, error)
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
func (usecase *StoreUseCase) Create(ctx context.Context, userID, name, description string, isActive bool) (*domain.Store, error) {
	store := domain.NewStore(userID, name, description, isActive)
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
func (usecase *StoreUseCase) Update(ctx context.Context, storeID string, name, description *string, isActive *bool) (*domain.Store, error) {
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

// GetScheduleByDay method retrieves the schedule of a store for a specific day
func (usecase *StoreUseCase) GetScheduleByDay(ctx context.Context, storeID string, dayOfWeek int32) (*domain.StoreSchedule, error) {
	schedule, err := usecase.scheduleRepo.GetStoreScheduleByDay(ctx, storeID, dayOfWeek)
	if err != nil {
		return nil, fmt.Errorf("getting store schedule: %w", err)
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

// StoreServiceRepository interface
type StoreServiceRepository interface {
	CreateStoreService(ctx context.Context, service *domain.StoreService) error
	GetStoreServiceByID(ctx context.Context, id string) (*domain.StoreService, error)
	ListStoreServicesByStoreID(ctx context.Context, storeID string) ([]*domain.StoreService, error)
	UpdateStoreService(ctx context.Context, service *domain.StoreService) error
	DeleteStoreService(ctx context.Context, id string) error
}

// StoreServiceUseCase struct
type StoreServiceUseCase struct {
	repo StoreServiceRepository
}

// NewStoreServiceUseCase constructor
func NewStoreServiceUseCase(repo StoreServiceRepository) *StoreServiceUseCase {
	return &StoreServiceUseCase{
		repo: repo,
	}
}

// Create method
func (usecase *StoreServiceUseCase) Create(ctx context.Context, storeID, name, description string, durationMinutes int32, price float64, isActive bool) (*domain.StoreService, error) {
	service := domain.NewStoreService(storeID, name, description, durationMinutes, price, isActive)

	if !service.IsValid() {
		return nil, domain.ErrInvalidInput
	}

	if err := usecase.repo.CreateStoreService(ctx, service); err != nil {
		return nil, fmt.Errorf("creating store service: %w", err)
	}

	return service, nil
}

// GetByID method
func (usecase *StoreServiceUseCase) GetByID(ctx context.Context, serviceID string) (*domain.StoreService, error) {
	service, err := usecase.repo.GetStoreServiceByID(ctx, serviceID)
	if err != nil {
		return nil, fmt.Errorf("getting store service: %w", err)
	}

	return service, nil
}

// ListByStoreID method
func (usecase *StoreServiceUseCase) ListByStoreID(ctx context.Context, storeID string) ([]*domain.StoreService, error) {
	services, err := usecase.repo.ListStoreServicesByStoreID(ctx, storeID)
	if err != nil {
		return nil, fmt.Errorf("listing store services by store id: %w", err)
	}

	return services, nil
}

// Update method
func (usecase *StoreServiceUseCase) Update(ctx context.Context, serviceID string, name, description *string, durationMinutes *int32, price *float64, isActive *bool) (*domain.StoreService, error) {
	service, err := usecase.repo.GetStoreServiceByID(ctx, serviceID)
	if err != nil {
		return nil, fmt.Errorf("getting store service to update: %w", err)
	}

	if name != nil {
		service.Name = *name
	}
	if description != nil {
		service.Description = *description
	}
	if durationMinutes != nil {
		service.DurationMinutes = *durationMinutes
	}
	if price != nil {
		service.Price = *price
	}
	if isActive != nil {
		service.IsActive = *isActive
	}

	if !service.IsValid() {
		return nil, domain.ErrInvalidInput
	}

	if err := usecase.repo.UpdateStoreService(ctx, service); err != nil {
		return nil, fmt.Errorf("updating store service: %w", err)
	}

	updatedService, err := usecase.repo.GetStoreServiceByID(ctx, serviceID)
	if err != nil {
		return nil, fmt.Errorf("getting updated store service: %w", err)
	}

	return updatedService, nil
}

// Delete method
func (usecase *StoreServiceUseCase) Delete(ctx context.Context, serviceID string) error {
	if err := usecase.repo.DeleteStoreService(ctx, serviceID); err != nil {
		return fmt.Errorf("deleting store service: %w", err)
	}

	return nil
}
