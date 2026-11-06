package usecase

import (
	"context"
	"fmt"

	"github.com/KevinManzano5/misto-backend/internal/domain"
)

// StoreCategoryRepository interface
type StoreCategoryRepository interface {
	ListStoreCategories(ctx context.Context) ([]*domain.StoreCategory, error)
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
