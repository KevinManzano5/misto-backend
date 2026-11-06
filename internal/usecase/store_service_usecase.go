package usecase

import (
	"context"
	"fmt"

	"github.com/KevinManzano5/misto-backend/internal/domain"
)

// StoreServiceRepository interface
type StoreServiceRepository interface {
	CreateStoreService(ctx context.Context, service *domain.StoreService) error
	GetStoreServiceByID(ctx context.Context, id string) (*domain.StoreService, error)
	ListStoreServicesByStore(ctx context.Context, storeID string) ([]*domain.StoreService, error)
	ListAllStoreServicesByStore(ctx context.Context, storeID string) ([]*domain.StoreService, error)
	UpdateStoreService(ctx context.Context, service *domain.StoreService) (*domain.StoreService, error)
	DeleteStoreService(ctx context.Context, id string) error
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
}

// NewStoreServiceUseCase constructor
func NewStoreServiceUseCase(repo StoreServiceRepository, storeRepo StoreServiceStoreRepository, categoryRepo StoreServiceCategoryRepository) *StoreServiceUseCase {
	return &StoreServiceUseCase{
		repo:         repo,
		storeRepo:    storeRepo,
		categoryRepo: categoryRepo,
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

// Create creates a new service for a store. Only the store's owner may call this, and
// categoryID must be one of the categories currently assigned to the store.
func (usecase *StoreServiceUseCase) Create(ctx context.Context, storeID, requesterUserID, categoryID, name, description string, durationMinutes int32, price float64) (*domain.StoreService, error) {
	if _, err := usecase.requireOwnedStore(ctx, storeID, requesterUserID); err != nil {
		return nil, err
	}

	if err := usecase.requireAssignedCategory(ctx, storeID, categoryID); err != nil {
		return nil, err
	}

	service := domain.NewStoreService(storeID, categoryID, name, description, durationMinutes, price)
	if !service.IsValid() {
		return nil, fmt.Errorf("%w: invalid store service data", domain.ErrInvalidInput)
	}

	if err := usecase.repo.CreateStoreService(ctx, service); err != nil {
		return nil, fmt.Errorf("creating store service: %w", err)
	}

	return service, nil
}

// Update applies partial changes to an existing service. Only the store's owner may call this,
// and a new categoryID (if given) must be one of the categories currently assigned to the store.
func (usecase *StoreServiceUseCase) Update(ctx context.Context, serviceID, requesterUserID string, categoryID, name, description *string, durationMinutes *int32, price *float64, isActive *bool) (*domain.StoreService, error) {
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
