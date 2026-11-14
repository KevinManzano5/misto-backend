package usecase

import (
	"context"
	"fmt"

	"github.com/KevinManzano5/misto-backend/internal/domain"
)

// StoreCategoryRepository interface
type StoreCategoryRepository interface {
	CreateStoreCategory(ctx context.Context, category *domain.StoreCategory) error
	GetStoreCategoryByID(ctx context.Context, id string) (*domain.StoreCategory, error)
	ListStoreCategories(ctx context.Context) ([]*domain.StoreCategory, error)
	UpdateStoreCategory(ctx context.Context, category *domain.StoreCategory) (*domain.StoreCategory, error)
	DeleteStoreCategory(ctx context.Context, id string) error
}

// StoreCategoryUseCase struct
type StoreCategoryUseCase struct {
	repo StoreCategoryRepository
}

// NewStoreCategoryUseCase constructor
func NewStoreCategoryUseCase(repo StoreCategoryRepository) *StoreCategoryUseCase {
	return &StoreCategoryUseCase{repo: repo}
}

// ListCategories returns the full catalog of store categories. Public: no authorization check.
func (usecase *StoreCategoryUseCase) ListCategories(ctx context.Context) ([]*domain.StoreCategory, error) {
	categories, err := usecase.repo.ListStoreCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing store categories: %w", err)
	}

	return categories, nil
}

// Create adds a new entry to the store category catalog. Admin-only, enforced by the router's
// admin middleware, not by this usecase.
func (usecase *StoreCategoryUseCase) Create(ctx context.Context, name, slug, description string) (*domain.StoreCategory, error) {
	category := domain.NewStoreCategory(name, slug, description)
	if !category.IsValid() {
		return nil, fmt.Errorf("%w: invalid store category data", domain.ErrInvalidInput)
	}

	if err := usecase.repo.CreateStoreCategory(ctx, category); err != nil {
		return nil, fmt.Errorf("creating store category: %w", err)
	}

	return category, nil
}

// Update applies partial changes to an existing catalog entry. Admin-only, enforced by the
// router's admin middleware, not by this usecase.
func (usecase *StoreCategoryUseCase) Update(ctx context.Context, id string, name, slug, description *string) (*domain.StoreCategory, error) {
	category, err := usecase.repo.GetStoreCategoryByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting store category: %w", err)
	}

	if name != nil {
		category.Name = *name
	}
	if slug != nil {
		category.Slug = *slug
	}
	if description != nil {
		category.Description = *description
	}

	if !category.IsValid() {
		return nil, fmt.Errorf("%w: invalid store category data", domain.ErrInvalidInput)
	}

	updated, err := usecase.repo.UpdateStoreCategory(ctx, category)
	if err != nil {
		return nil, fmt.Errorf("updating store category: %w", err)
	}

	return updated, nil
}

// Delete removes an entry from the store category catalog. Admin-only, enforced by the router's
// admin middleware, not by this usecase. Fails if the category is still assigned to a store or
// used by a service.
func (usecase *StoreCategoryUseCase) Delete(ctx context.Context, id string) error {
	if _, err := usecase.repo.GetStoreCategoryByID(ctx, id); err != nil {
		return fmt.Errorf("getting store category: %w", err)
	}

	if err := usecase.repo.DeleteStoreCategory(ctx, id); err != nil {
		return fmt.Errorf("deleting store category: %w", err)
	}

	return nil
}
