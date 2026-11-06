package postgres

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/KevinManzano5/misto-backend/internal/domain"
	postgres "github.com/KevinManzano5/misto-backend/internal/repository/postgres/generated"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StoreServiceRepository struct
type StoreServiceRepository struct {
	pool    *pgxpool.Pool
	queries *postgres.Queries
}

// NewStoreServiceRepository constructor
func NewStoreServiceRepository(pool *pgxpool.Pool) *StoreServiceRepository {
	return &StoreServiceRepository{
		pool:    pool,
		queries: postgres.New(pool),
	}
}

// toDomainStoreService function
func toDomainStoreService(service postgres.StoreService) (*domain.StoreService, error) {
	var description string
	if service.Description != nil {
		description = *service.Description
	}

	price, err := service.Price.Float64Value()
	if err != nil {
		return nil, fmt.Errorf("decoding store service price: %w", err)
	}

	return &domain.StoreService{
		ID:                 service.ID.String(),
		StoreID:            service.StoreID.String(),
		CategoryID:         service.CategoryID.String(),
		Name:               service.Name,
		Description:        description,
		DurationMinutes:    service.DurationMinutes,
		Price:              price.Float64,
		IsActive:           service.IsActive,
		MinLeadTimeMinutes: service.MinLeadTimeMinutes,
		CreatedAt:          service.CreatedAt.Time.UTC(),
		UpdatedAt:          service.UpdatedAt.Time.UTC(),
	}, nil
}

// toPgNumericPrice converts a domain price (float64) into the pgtype.Numeric sqlc expects.
func toPgNumericPrice(price float64) (pgtype.Numeric, error) {
	var numeric pgtype.Numeric

	if err := numeric.Scan(strconv.FormatFloat(price, 'f', 2, 64)); err != nil {
		return pgtype.Numeric{}, fmt.Errorf("invalid price: %w", domain.ErrInvalidInput)
	}

	return numeric, nil
}

// translateStoreServiceWriteError maps known Postgres constraint violations from writing a
// store service to the shared domain sentinel errors.
func translateStoreServiceWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" { // foreign_key_violation: store_id or category_id does not reference an existing row
		return fmt.Errorf("%w: referenced store or category does not exist", domain.ErrInvalidInput)
	}

	return fmt.Errorf("writing store service: %w", err)
}

// CreateStoreService method
func (repo *StoreServiceRepository) CreateStoreService(ctx context.Context, service *domain.StoreService) error {
	var storeID, categoryID pgtype.UUID

	if err := storeID.Scan(service.StoreID); err != nil {
		return fmt.Errorf("invalid storeID: %w", domain.ErrInvalidInput)
	}
	if err := categoryID.Scan(service.CategoryID); err != nil {
		return fmt.Errorf("invalid categoryID: %w", domain.ErrInvalidInput)
	}

	price, err := toPgNumericPrice(service.Price)
	if err != nil {
		return err
	}

	var description *string
	if service.Description != "" {
		description = &service.Description
	}

	result, err := repo.queries.CreateStoreService(ctx, postgres.CreateStoreServiceParams{
		StoreID:            storeID,
		CategoryID:         categoryID,
		Name:               service.Name,
		Description:        description,
		DurationMinutes:    service.DurationMinutes,
		Price:              price,
		MinLeadTimeMinutes: service.MinLeadTimeMinutes,
	})
	if err != nil {
		return translateStoreServiceWriteError(err)
	}

	created, err := toDomainStoreService(result)
	if err != nil {
		return err
	}

	*service = *created

	return nil
}

// GetStoreServiceByID method
func (repo *StoreServiceRepository) GetStoreServiceByID(ctx context.Context, id string) (*domain.StoreService, error) {
	var serviceID pgtype.UUID

	if err := serviceID.Scan(id); err != nil {
		return nil, fmt.Errorf("invalid serviceID: %w", domain.ErrInvalidInput)
	}

	result, err := repo.queries.GetStoreServiceByID(ctx, serviceID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("store service with id %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("getting store service %s: %w", id, err)
	}

	return toDomainStoreService(result)
}

// ListStoreServicesByStore method returns the active services offered by a store, ordered by name.
func (repo *StoreServiceRepository) ListStoreServicesByStore(ctx context.Context, storeID string) ([]*domain.StoreService, error) {
	var sID pgtype.UUID

	if err := sID.Scan(storeID); err != nil {
		return nil, fmt.Errorf("invalid storeID: %w", domain.ErrInvalidInput)
	}

	results, err := repo.queries.ListStoreServicesByStore(ctx, sID)
	if err != nil {
		return nil, fmt.Errorf("listing store services: %w", err)
	}

	return toDomainStoreServices(results)
}

// ListAllStoreServicesByStore method returns every service (active or not) offered by a store,
// ordered by name, for the store owner's own management view.
func (repo *StoreServiceRepository) ListAllStoreServicesByStore(ctx context.Context, storeID string) ([]*domain.StoreService, error) {
	var sID pgtype.UUID

	if err := sID.Scan(storeID); err != nil {
		return nil, fmt.Errorf("invalid storeID: %w", domain.ErrInvalidInput)
	}

	results, err := repo.queries.ListAllStoreServicesByStore(ctx, sID)
	if err != nil {
		return nil, fmt.Errorf("listing store services: %w", err)
	}

	return toDomainStoreServices(results)
}

// toDomainStoreServices function
func toDomainStoreServices(results []postgres.StoreService) ([]*domain.StoreService, error) {
	services := make([]*domain.StoreService, 0, len(results))
	for _, result := range results {
		service, err := toDomainStoreService(result)
		if err != nil {
			return nil, err
		}
		services = append(services, service)
	}

	return services, nil
}

// UpdateStoreService method
func (repo *StoreServiceRepository) UpdateStoreService(ctx context.Context, service *domain.StoreService) (*domain.StoreService, error) {
	var serviceID, categoryID pgtype.UUID

	if err := serviceID.Scan(service.ID); err != nil {
		return nil, fmt.Errorf("invalid serviceID: %w", domain.ErrInvalidInput)
	}
	if err := categoryID.Scan(service.CategoryID); err != nil {
		return nil, fmt.Errorf("invalid categoryID: %w", domain.ErrInvalidInput)
	}

	price, err := toPgNumericPrice(service.Price)
	if err != nil {
		return nil, err
	}

	var description *string
	if service.Description != "" {
		description = &service.Description
	}

	result, err := repo.queries.UpdateStoreService(ctx, postgres.UpdateStoreServiceParams{
		ID:                 serviceID,
		CategoryID:         categoryID,
		Name:               service.Name,
		Description:        description,
		DurationMinutes:    service.DurationMinutes,
		Price:              price,
		IsActive:           service.IsActive,
		MinLeadTimeMinutes: service.MinLeadTimeMinutes,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("store service with id %s: %w", service.ID, domain.ErrNotFound)
		}
		return nil, translateStoreServiceWriteError(err)
	}

	return toDomainStoreService(result)
}

// DeleteStoreService method
func (repo *StoreServiceRepository) DeleteStoreService(ctx context.Context, id string) error {
	var serviceID pgtype.UUID

	if err := serviceID.Scan(id); err != nil {
		return fmt.Errorf("invalid serviceID: %w", domain.ErrInvalidInput)
	}

	if err := repo.queries.DeleteStoreService(ctx, serviceID); err != nil {
		return fmt.Errorf("deleting store service %s: %w", id, err)
	}

	return nil
}
