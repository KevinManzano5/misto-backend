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

// StoreRepository struct
type StoreRepository struct {
	pool    *pgxpool.Pool
	queries *postgres.Queries
}

// toDomainStore function
func toDomainStore(store postgres.Store) *domain.Store {
	return &domain.Store{
		ID:          store.ID.String(),
		UserID:      store.UserID.String(),
		Name:        store.Name,
		Description: *store.Description,
		IsActive:    store.IsActive,
		CreatedAt:   store.CreatedAt.Time,
		UpdatedAt:   store.UpdatedAt.Time,
	}
}

// NewStoreRepository function
func NewStoreRepository(pool *pgxpool.Pool) *StoreRepository {
	return &StoreRepository{
		pool:    pool,
		queries: postgres.New(pool),
	}
}

// CreateStore method
func (repo *StoreRepository) CreateStore(ctx context.Context, store *domain.Store) error {
	var userID pgtype.UUID

	if err := userID.Scan(store.UserID); err != nil {
		return fmt.Errorf("invalid created_by: %w", domain.ErrInvalidInput)
	}

	result, err := repo.queries.CreateStore(ctx, postgres.CreateStoreParams{
		UserID:      userID,
		Name:        store.Name,
		Description: &store.Description,
	})

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("%w", domain.ErrAlreadyExists)
		}

		return fmt.Errorf("creating store: %w", err)
	}

	store.ID = result.ID.String()
	store.CreatedAt = result.CreatedAt.Time
	store.UpdatedAt = result.UpdatedAt.Time

	return nil
}

// GetStoreByID method
func (repo *StoreRepository) GetStoreByID(ctx context.Context, id string) (*domain.Store, error) {
	var storeID pgtype.UUID

	if err := storeID.Scan(id); err != nil {
		return nil, fmt.Errorf("invalid storeID: %w", domain.ErrInvalidInput)
	}

	result, err := repo.queries.GetStoreByID(ctx, storeID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("store with id %s: %w", id, domain.ErrNotFound)
		}

		return nil, fmt.Errorf("getting store %s: %w", id, err)
	}

	return toDomainStore(result), nil
}

// GetStoreByUserID method
func (repo *StoreRepository) GetStoreByUserID(ctx context.Context, id string) (*domain.Store, error) {
	var userID pgtype.UUID

	if err := userID.Scan(id); err != nil {
		return nil, fmt.Errorf("invalid userID: %w", domain.ErrInvalidInput)
	}

	result, err := repo.queries.GetStoreByUserID(ctx, userID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("store with userID %s: %w", id, domain.ErrNotFound)
		}

		return nil, fmt.Errorf("getting store %s: %w", id, err)
	}

	return toDomainStore(result), nil
}

// UpdateStore method
func (repo *StoreRepository) UpdateStore(ctx context.Context, store *domain.Store) error {
	var storeID pgtype.UUID

	if err := storeID.Scan(store.ID); err != nil {
		return fmt.Errorf("invalid storeID: %w", domain.ErrInvalidInput)
	}

	_, err := repo.queries.UpdateStore(ctx, postgres.UpdateStoreParams{
		ID:          storeID,
		Name:        store.Name,
		Description: &store.Description,
		IsActive:    store.IsActive,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("store with id %s: %w", storeID, domain.ErrNotFound)
		}

		return fmt.Errorf("searching store %s: %w", storeID, err)
	}

	return nil
}

// DeleteStore method
func (repo *StoreRepository) DeleteStore(ctx context.Context, id string) error {
	var storeID pgtype.UUID

	if err := storeID.Scan(id); err != nil {
		return fmt.Errorf("invalid storeID: %w", domain.ErrInvalidInput)
	}

	err := repo.queries.DeleteStore(ctx, storeID)

	if err != nil {
		return fmt.Errorf("deleting store: %w", err)
	}

	return nil
}
