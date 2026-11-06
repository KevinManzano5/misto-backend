package postgres

import (
	"context"
	"fmt"

	"github.com/KevinManzano5/misto-backend/internal/domain"
	postgres "github.com/KevinManzano5/misto-backend/internal/repository/postgres/generated"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StoreCategoryRepository struct
type StoreCategoryRepository struct {
	pool    *pgxpool.Pool
	queries *postgres.Queries
}

// NewStoreCategoryRepository constructor
func NewStoreCategoryRepository(pool *pgxpool.Pool) *StoreCategoryRepository {
	return &StoreCategoryRepository{
		pool:    pool,
		queries: postgres.New(pool),
	}
}

// toDomainStoreCategory function
func toDomainStoreCategory(category postgres.StoreCategory) *domain.StoreCategory {
	var description string
	if category.Description != nil {
		description = *category.Description
	}

	return &domain.StoreCategory{
		ID:          category.ID.String(),
		Name:        category.Name,
		Slug:        category.Slug,
		Description: description,
		CreatedAt:   category.CreatedAt.Time.UTC(),
		UpdatedAt:   category.UpdatedAt.Time.UTC(),
	}
}

// ListStoreCategories method returns the full catalog of store categories, ordered by name.
func (repo *StoreCategoryRepository) ListStoreCategories(ctx context.Context) ([]*domain.StoreCategory, error) {
	results, err := repo.queries.ListStoreCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing store categories: %w", err)
	}

	categories := make([]*domain.StoreCategory, 0, len(results))
	for _, result := range results {
		categories = append(categories, toDomainStoreCategory(result))
	}

	return categories, nil
}
