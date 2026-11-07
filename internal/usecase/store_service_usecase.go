package usecase

import (
	"context"
	"fmt"

	"github.com/KevinManzano5/misto-backend/internal/domain"
)

// defaultOpenTime and defaultCloseTime are the opening hours applied to every day of the week
// when a store service is created, until the owner customizes its schedule.
const (
	defaultOpenTime  = "09:00"
	defaultCloseTime = "18:00"
)

// sunday is the domain.StoreServiceSchedule.DayOfWeek value for Sunday, closed by default.
const sunday = 0

// StoreServiceRepository interface
type StoreServiceRepository interface {
	CreateStoreService(ctx context.Context, service *domain.StoreService) error
	CreateStoreServiceWithSchedules(ctx context.Context, service *domain.StoreService, schedules []*domain.StoreServiceSchedule) error
	GetStoreServiceByID(ctx context.Context, id string) (*domain.StoreService, error)
	ListStoreServicesByStore(ctx context.Context, storeID string) ([]*domain.StoreService, error)
	ListAllStoreServicesByStore(ctx context.Context, storeID string) ([]*domain.StoreService, error)
	UpdateStoreService(ctx context.Context, service *domain.StoreService) (*domain.StoreService, error)
	DeleteStoreService(ctx context.Context, id string) error
}

// StoreServiceScheduleRepository interface (narrowed to what StoreServiceUseCase needs to manage
// a service's weekly schedule)
type StoreServiceScheduleRepository interface {
	UpsertStoreServiceSchedule(ctx context.Context, schedule *domain.StoreServiceSchedule) error
	ListStoreServiceSchedules(ctx context.Context, storeServiceID string) ([]*domain.StoreServiceSchedule, error)
}

// StoreServiceStoreRepository interface (narrowed to what StoreServiceUseCase needs from a store)
type StoreServiceStoreRepository interface {
	GetStoreByID(ctx context.Context, storeID string) (*domain.Store, error)
}

// StoreServiceCategoryRepository interface (narrowed to what StoreServiceUseCase needs to check
// category membership)
type StoreServiceCategoryRepository interface {
	ListCategoriesForStore(ctx context.Context, storeID string) ([]*domain.StoreCategory, error)
}

// StoreServiceUseCase struct
type StoreServiceUseCase struct {
	repo         StoreServiceRepository
	storeRepo    StoreServiceStoreRepository
	categoryRepo StoreServiceCategoryRepository
	scheduleRepo StoreServiceScheduleRepository
}

// NewStoreServiceUseCase constructor
func NewStoreServiceUseCase(repo StoreServiceRepository, storeRepo StoreServiceStoreRepository, categoryRepo StoreServiceCategoryRepository, scheduleRepo StoreServiceScheduleRepository) *StoreServiceUseCase {
	return &StoreServiceUseCase{
		repo:         repo,
		storeRepo:    storeRepo,
		categoryRepo: categoryRepo,
		scheduleRepo: scheduleRepo,
	}
}

// requireOwnedStore loads a store and ensures requesterUserID owns it.
func (usecase *StoreServiceUseCase) requireOwnedStore(ctx context.Context, storeID, requesterUserID string) (*domain.Store, error) {
	store, err := usecase.storeRepo.GetStoreByID(ctx, storeID)
	if err != nil {
		return nil, fmt.Errorf("getting store: %w", err)
	}

	if store.UserID != requesterUserID {
		return nil, fmt.Errorf("%w: only the store owner can manage its services", domain.ErrForbidden)
	}

	return store, nil
}

// requireAssignedCategory ensures categoryID is one of the categories currently assigned to storeID.
func (usecase *StoreServiceUseCase) requireAssignedCategory(ctx context.Context, storeID, categoryID string) error {
	categories, err := usecase.categoryRepo.ListCategoriesForStore(ctx, storeID)
	if err != nil {
		return fmt.Errorf("listing store categories: %w", err)
	}

	for _, category := range categories {
		if category.ID == categoryID {
			return nil
		}
	}

	return fmt.Errorf("%w: category is not assigned to this store", domain.ErrInvalidInput)
}

// Create creates a new service for a store, along with a default weekly schedule (open every
// day 09:00-18:00, except Sunday which is closed), which the owner can later customize. Only the
// store's owner may call this, and categoryID must be one of the categories currently assigned
// to the store.
func (usecase *StoreServiceUseCase) Create(ctx context.Context, storeID, requesterUserID, categoryID, name, description string, durationMinutes int32, price float64, minLeadTimeMinutes int32) (*domain.StoreService, error) {
	if _, err := usecase.requireOwnedStore(ctx, storeID, requesterUserID); err != nil {
		return nil, err
	}

	if err := usecase.requireAssignedCategory(ctx, storeID, categoryID); err != nil {
		return nil, err
	}

	service := domain.NewStoreService(storeID, categoryID, name, description, durationMinutes, price, minLeadTimeMinutes)
	if !service.IsValid() {
		return nil, fmt.Errorf("%w: invalid store service data", domain.ErrInvalidInput)
	}

	schedules := defaultWeeklyServiceSchedules()

	if err := usecase.repo.CreateStoreServiceWithSchedules(ctx, service, schedules); err != nil {
		return nil, fmt.Errorf("creating store service: %w", err)
	}

	return service, nil
}

// defaultWeeklyServiceSchedules builds the default opening-hours schedule for all seven days of
// the week, applied to every newly created store service.
func defaultWeeklyServiceSchedules() []*domain.StoreServiceSchedule {
	schedules := make([]*domain.StoreServiceSchedule, 0, 7)

	for dayOfWeek := int32(0); dayOfWeek <= 6; dayOfWeek++ {
		isClosed := dayOfWeek == sunday

		var segments []domain.StoreServiceScheduleSegment
		if !isClosed {
			segments = []domain.StoreServiceScheduleSegment{{OpenTime: defaultOpenTime, CloseTime: defaultCloseTime}}
		}

		schedules = append(schedules, domain.NewStoreServiceSchedule("", dayOfWeek, segments, isClosed))
	}

	return schedules
}

// Update applies partial changes to an existing service. Only the store's owner may call this,
// and a new categoryID (if given) must be one of the categories currently assigned to the store.
func (usecase *StoreServiceUseCase) Update(ctx context.Context, serviceID, requesterUserID string, categoryID, name, description *string, durationMinutes *int32, price *float64, isActive *bool, minLeadTimeMinutes *int32) (*domain.StoreService, error) {
	service, err := usecase.repo.GetStoreServiceByID(ctx, serviceID)
	if err != nil {
		return nil, fmt.Errorf("getting store service: %w", err)
	}

	if _, err := usecase.requireOwnedStore(ctx, service.StoreID, requesterUserID); err != nil {
		return nil, err
	}

	if categoryID != nil {
		if err := usecase.requireAssignedCategory(ctx, service.StoreID, *categoryID); err != nil {
			return nil, err
		}
		service.CategoryID = *categoryID
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
	if minLeadTimeMinutes != nil {
		service.MinLeadTimeMinutes = *minLeadTimeMinutes
	}

	if !service.IsValid() {
		return nil, fmt.Errorf("%w: invalid store service data", domain.ErrInvalidInput)
	}

	updated, err := usecase.repo.UpdateStoreService(ctx, service)
	if err != nil {
		return nil, fmt.Errorf("updating store service: %w", err)
	}

	return updated, nil
}

// Delete removes a service. Only the store's owner may call this.
func (usecase *StoreServiceUseCase) Delete(ctx context.Context, serviceID, requesterUserID string) error {
	service, err := usecase.repo.GetStoreServiceByID(ctx, serviceID)
	if err != nil {
		return fmt.Errorf("getting store service: %w", err)
	}

	if _, err := usecase.requireOwnedStore(ctx, service.StoreID, requesterUserID); err != nil {
		return err
	}

	if err := usecase.repo.DeleteStoreService(ctx, serviceID); err != nil {
		return fmt.Errorf("deleting store service: %w", err)
	}

	return nil
}

// ListByStore returns the active services offered by a store. Public: no authorization check.
func (usecase *StoreServiceUseCase) ListByStore(ctx context.Context, storeID string) ([]*domain.StoreService, error) {
	services, err := usecase.repo.ListStoreServicesByStore(ctx, storeID)
	if err != nil {
		return nil, fmt.Errorf("listing store services: %w", err)
	}

	return services, nil
}

// UpsertSchedule method creates or updates the schedule of a service for a specific day. Only
// the store's owner may call this.
func (usecase *StoreServiceUseCase) UpsertSchedule(ctx context.Context, serviceID, requesterUserID string, dayOfWeek int32, segments []domain.StoreServiceScheduleSegment, isClosed bool) (*domain.StoreServiceSchedule, error) {
	service, err := usecase.repo.GetStoreServiceByID(ctx, serviceID)
	if err != nil {
		return nil, fmt.Errorf("getting store service: %w", err)
	}

	if _, err := usecase.requireOwnedStore(ctx, service.StoreID, requesterUserID); err != nil {
		return nil, err
	}

	schedule := domain.NewStoreServiceSchedule(serviceID, dayOfWeek, segments, isClosed)

	if !schedule.IsValid() {
		return nil, domain.ErrInvalidInput
	}

	if err := usecase.scheduleRepo.UpsertStoreServiceSchedule(ctx, schedule); err != nil {
		return nil, fmt.Errorf("upserting store service schedule: %w", err)
	}

	return schedule, nil
}

// ListSchedules method retrieves all schedules for a service. Public: no authorization check.
func (usecase *StoreServiceUseCase) ListSchedules(ctx context.Context, serviceID string) ([]*domain.StoreServiceSchedule, error) {
	schedules, err := usecase.scheduleRepo.ListStoreServiceSchedules(ctx, serviceID)
	if err != nil {
		return nil, fmt.Errorf("listing store service schedules: %w", err)
	}

	return schedules, nil
}
