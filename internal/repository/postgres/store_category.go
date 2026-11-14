package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/KevinManzano5/misto-backend/internal/domain"
	postgres "github.com/KevinManzano5/misto-backend/internal/repository/postgres/generated"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
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

// translateStoreCategoryWriteError maps known Postgres constraint violations from writing a
// store category to the shared domain sentinel errors.
func translateStoreCategoryWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation: name or slug already taken
		return fmt.Errorf("%w: a category with this name or slug already exists", domain.ErrAlreadyExists)
	}

	return fmt.Errorf("writing store category: %w", err)
}

// CreateStoreCategory method
func (repo *StoreCategoryRepository) CreateStoreCategory(ctx context.Context, category *domain.StoreCategory) error {
	var description *string
	if category.Description != "" {
		description = &category.Description
	}

	result, err := repo.queries.CreateStoreCategory(ctx, postgres.CreateStoreCategoryParams{
		Name:        category.Name,
		Slug:        category.Slug,
		Description: description,
	})
	if err != nil {
		return translateStoreCategoryWriteError(err)
	}

	*category = *toDomainStoreCategory(result)

	return nil
}

// GetStoreCategoryByID method
func (repo *StoreCategoryRepository) GetStoreCategoryByID(ctx context.Context, id string) (*domain.StoreCategory, error) {
	var categoryID pgtype.UUID

	if err := categoryID.Scan(id); err != nil {
		return nil, fmt.Errorf("invalid categoryID: %w", domain.ErrInvalidInput)
	}

	result, err := repo.queries.GetStoreCategoryByID(ctx, categoryID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("store category with id %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("getting store category %s: %w", id, err)
	}

	return toDomainStoreCategory(result), nil
}

// UpdateStoreCategory method
func (repo *StoreCategoryRepository) UpdateStoreCategory(ctx context.Context, category *domain.StoreCategory) (*domain.StoreCategory, error) {
	var categoryID pgtype.UUID

	if err := categoryID.Scan(category.ID); err != nil {
		return nil, fmt.Errorf("invalid categoryID: %w", domain.ErrInvalidInput)
	}

	var description *string
	if category.Description != "" {
		description = &category.Description
	}

	result, err := repo.queries.UpdateStoreCategory(ctx, postgres.UpdateStoreCategoryParams{
		ID:          categoryID,
		Name:        category.Name,
		Slug:        category.Slug,
		Description: description,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("store category with id %s: %w", category.ID, domain.ErrNotFound)
		}
		return nil, translateStoreCategoryWriteError(err)
	}

	return toDomainStoreCategory(result), nil
}

// DeleteStoreCategory method
func (repo *StoreCategoryRepository) DeleteStoreCategory(ctx context.Context, id string) error {
	var categoryID pgtype.UUID

	if err := categoryID.Scan(id); err != nil {
		return fmt.Errorf("invalid categoryID: %w", domain.ErrInvalidInput)
	}

	if err := repo.queries.DeleteStoreCategory(ctx, categoryID); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" { // foreign_key_violation: category is still assigned to a store or used by a service
			return fmt.Errorf("%w: category is in use by one or more stores and cannot be deleted", domain.ErrAlreadyExists)
		}
		return fmt.Errorf("deleting store category %s: %w", id, err)
	}

	return nil
}
