package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/KevinManzano5/misto-backend/internal/domain"
	postgres "github.com/KevinManzano5/misto-backend/internal/repository/postgres/generated"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StoreCategoryAssignmentRepository struct
type StoreCategoryAssignmentRepository struct {
	pool    *pgxpool.Pool
	queries *postgres.Queries
}

// NewStoreCategoryAssignmentRepository constructor
func NewStoreCategoryAssignmentRepository(pool *pgxpool.Pool) *StoreCategoryAssignmentRepository {
	return &StoreCategoryAssignmentRepository{
		pool:    pool,
		queries: postgres.New(pool),
	}
}

// translateStoreCategoryAssignmentWriteError maps known Postgres constraint violations from
// assigning categories to a store to the shared domain sentinel errors.
func translateStoreCategoryAssignmentWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" { // foreign_key_violation: category_id does not reference an existing row
		return fmt.Errorf("%w: referenced category does not exist", domain.ErrInvalidInput)
	}

	return fmt.Errorf("writing store category assignment: %w", err)
}

// ListCategoriesForStore method returns the categories currently assigned to a store, ordered by name.
func (repo *StoreCategoryAssignmentRepository) ListCategoriesForStore(ctx context.Context, storeID string) ([]*domain.StoreCategory, error) {
	var sID pgtype.UUID

	if err := sID.Scan(storeID); err != nil {
		return nil, fmt.Errorf("invalid storeID: %w", domain.ErrInvalidInput)
	}

	results, err := repo.queries.ListCategoriesForStore(ctx, sID)
	if err != nil {
		return nil, fmt.Errorf("listing categories for store: %w", err)
	}

	categories := make([]*domain.StoreCategory, 0, len(results))
	for _, result := range results {
		categories = append(categories, toDomainStoreCategory(result))
	}

	return categories, nil
}

// ReplaceStoreCategories method atomically replaces the full set of categories assigned to a
// store: it deletes every existing assignment for the store and inserts the given ones, so a
// failure inserting any category rolls back the deletion instead of leaving the store with no
// categories assigned.
func (repo *StoreCategoryAssignmentRepository) ReplaceStoreCategories(ctx context.Context, storeID string, categoryIDs []string) error {
	var sID pgtype.UUID

	if err := sID.Scan(storeID); err != nil {
		return fmt.Errorf("invalid storeID: %w", domain.ErrInvalidInput)
	}

	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning store category assignment transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txQueries := repo.queries.WithTx(tx)

	if err := txQueries.DeleteStoreCategoryAssignments(ctx, sID); err != nil {
		return fmt.Errorf("clearing store category assignments: %w", err)
	}

	for _, categoryID := range categoryIDs {
		var cID pgtype.UUID
		if err := cID.Scan(categoryID); err != nil {
			return fmt.Errorf("invalid categoryID %q: %w", categoryID, domain.ErrInvalidInput)
		}

		if err := txQueries.InsertStoreCategoryAssignment(ctx, postgres.InsertStoreCategoryAssignmentParams{
			StoreID:    sID,
			CategoryID: cID,
		}); err != nil {
			return translateStoreCategoryAssignmentWriteError(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing store category assignments: %w", err)
	}

	return nil
}
