package usecase

import (
	"context"
	"fmt"

	"github.com/KevinManzano5/misto-backend/internal/domain"
)

// maxCategoriesPerStore caps how many categories a store may be assigned at once.
const maxCategoriesPerStore = 5

// StoreCategoryAssignmentRepository interface
type StoreCategoryAssignmentRepository interface {
	ListCategoriesForStore(ctx context.Context, storeID string) ([]*domain.StoreCategory, error)
	ReplaceStoreCategories(ctx context.Context, storeID string, categoryIDs []string) error
}

// StoreCategoryAssignmentStoreRepository interface (narrowed to what StoreCategoryAssignmentUseCase needs from a store)
type StoreCategoryAssignmentStoreRepository interface {
	GetStoreByID(ctx context.Context, storeID string) (*domain.Store, error)
}

// StoreCategoryAssignmentUseCase struct
type StoreCategoryAssignmentUseCase struct {
	repo      StoreCategoryAssignmentRepository
	storeRepo StoreCategoryAssignmentStoreRepository
}

// NewStoreCategoryAssignmentUseCase constructor
func NewStoreCategoryAssignmentUseCase(repo StoreCategoryAssignmentRepository, storeRepo StoreCategoryAssignmentStoreRepository) *StoreCategoryAssignmentUseCase {
	return &StoreCategoryAssignmentUseCase{
		repo:      repo,
		storeRepo: storeRepo,
	}
}

// ListStoreCategories returns the categories currently assigned to a store. Public: no
// authorization check.
func (usecase *StoreCategoryAssignmentUseCase) ListStoreCategories(ctx context.Context, storeID string) ([]*domain.StoreCategory, error) {
	categories, err := usecase.repo.ListCategoriesForStore(ctx, storeID)
	if err != nil {
		return nil, fmt.Errorf("listing store categories: %w", err)
	}

	return categories, nil
}

// ReplaceCategories replaces the full set of categories assigned to a store. Only the store's
// owner may call this, and at most maxCategoriesPerStore distinct categories may be assigned.
func (usecase *StoreCategoryAssignmentUseCase) ReplaceCategories(ctx context.Context, storeID, requesterUserID string, categoryIDs []string) ([]*domain.StoreCategory, error) {
	store, err := usecase.storeRepo.GetStoreByID(ctx, storeID)
	if err != nil {
		return nil, fmt.Errorf("getting store: %w", err)
	}

	if store.UserID != requesterUserID {
		return nil, fmt.Errorf("%w: only the store owner can assign its categories", domain.ErrForbidden)
	}

	uniqueIDs := dedupeStrings(categoryIDs)
	if len(uniqueIDs) > maxCategoriesPerStore {
		return nil, fmt.Errorf("%w: a store may have at most %d categories", domain.ErrInvalidInput, maxCategoriesPerStore)
	}

	if err := usecase.repo.ReplaceStoreCategories(ctx, storeID, uniqueIDs); err != nil {
		return nil, fmt.Errorf("replacing store categories: %w", err)
	}

	categories, err := usecase.repo.ListCategoriesForStore(ctx, storeID)
	if err != nil {
		return nil, fmt.Errorf("listing updated store categories: %w", err)
	}

	return categories, nil
}

// dedupeStrings returns values with duplicates removed, preserving first-seen order.
func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))

	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}
