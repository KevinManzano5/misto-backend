package usecase

import (
	"context"
	"fmt"

	"github.com/KevinManzano5/misto-backend/internal/domain"
)

// StoreRepository interface
type StoreRepository interface {
	CreateStore(ctx context.Context, store *domain.Store) error
	GetStoreByID(ctx context.Context, storeID string) (*domain.Store, error)
	GetStoreByUserID(ctx context.Context, userID string) (*domain.Store, error)
	UpdateStore(ctx context.Context, store *domain.Store) error
	DeleteStore(ctx context.Context, id string) error
}

// StoreUseCase struct
type StoreUseCase struct {
	repo StoreRepository
}

// NewStoreUseCase constructor
func NewStoreUseCase(repo StoreRepository) *StoreUseCase {
	return &StoreUseCase{
		repo: repo,
	}
}

// Create method
func (usecase *StoreUseCase) Create(ctx context.Context, userID, name, description string, isActive bool) (*domain.Store, error) {
	store := domain.NewStore(userID, name, description, isActive)

	if err := usecase.repo.CreateStore(ctx, store); err != nil {
		return nil, fmt.Errorf("creating store: %w", err)
	}

	return store, nil
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
