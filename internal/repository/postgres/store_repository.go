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

// ListStores method
func (repo *StoreRepository) ListStores(ctx context.Context) ([]*domain.Store, error) {
	results, err := repo.queries.ListStoresPaginated(ctx, postgres.ListStoresPaginatedParams{
		Limit:  10,
		Offset: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("listing stores: %w", err)
	}

	var stores []*domain.Store
	for _, store := range results {
		stores = append(stores, toDomainStore(store))
	}

	// If there are no results, we return an initialized empty slice instead of nil.
	if stores == nil {
		stores = make([]*domain.Store, 0)
	}

	return stores, nil
}

// StoreServiceRepository struct
type StoreServiceRepository struct {
	pool    *pgxpool.Pool
	queries *postgres.Queries
}

// NewStoreServiceRepository function
func NewStoreServiceRepository(pool *pgxpool.Pool) *StoreServiceRepository {
	return &StoreServiceRepository{
		pool:    pool,
		queries: postgres.New(pool),
	}
}

// toDomainStoreService function
func toDomainStoreService(service postgres.StoreService) *domain.StoreService {
	// Converting pgtype.Numeric to float64 (Assuming your domain uses float64 for Price)
	var price float64
	if service.Price.Valid {
		fV, _ := service.Price.Float64Value()
		price = fV.Float64
	}

	var desc string
	if service.Description != nil {
		desc = *service.Description
	}

	return &domain.StoreService{
		ID:              service.ID.String(),
		StoreID:         service.StoreID.String(),
		Name:            service.Name,
		Description:     desc,
		Price:           price,
		DurationMinutes: service.DurationMinutes,
		IsActive:        service.IsActive,
		CreatedAt:       service.CreatedAt.Time,
		UpdatedAt:       service.UpdatedAt.Time,
	}
}

// CreateStoreService method
func (repo *StoreServiceRepository) CreateStoreService(ctx context.Context, service *domain.StoreService) error {
	var storeID pgtype.UUID
	if err := storeID.Scan(service.StoreID); err != nil {
		return fmt.Errorf("invalid store_id: %w", domain.ErrInvalidInput)
	}

	// Configure Numeric for the price
	var price pgtype.Numeric
	if err := price.Scan(fmt.Sprintf("%f", service.Price)); err != nil {
		return fmt.Errorf("invalid price: %w", err)
	}

	result, err := repo.queries.CreateStoreService(ctx, postgres.CreateStoreServiceParams{
		StoreID:         storeID,
		Name:            service.Name,
		Description:     &service.Description,
		Price:           price,
		DurationMinutes: service.DurationMinutes,
	})

	if err != nil {
		return fmt.Errorf("creating store service: %w", err)
	}

	service.ID = result.ID.String()
	service.CreatedAt = result.CreatedAt.Time
	service.UpdatedAt = result.UpdatedAt.Time

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
			return nil, fmt.Errorf("service with id %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("getting store service %s: %w", id, err)
	}

	return toDomainStoreService(result), nil
}

// ListStoreServicesByStoreID method
func (repo *StoreServiceRepository) ListStoreServicesByStoreID(ctx context.Context, storeID string) ([]*domain.StoreService, error) {
	var sID pgtype.UUID

	if err := sID.Scan(storeID); err != nil {
		return nil, fmt.Errorf("invalid storeID: %w", domain.ErrInvalidInput)
	}

	results, err := repo.queries.ListStoreServicesByStoreID(ctx, sID)
	if err != nil {
		return nil, fmt.Errorf("listing store services: %w", err)
	}

	services := make([]*domain.StoreService, 0)
	for _, s := range results {
		services = append(services, toDomainStoreService(s))
	}

	return services, nil
}

// UpdateStoreService method
func (repo *StoreServiceRepository) UpdateStoreService(ctx context.Context, service *domain.StoreService) error {
	var serviceID pgtype.UUID
	if err := serviceID.Scan(service.ID); err != nil {
		return fmt.Errorf("invalid serviceID: %w", domain.ErrInvalidInput)
	}

	var price pgtype.Numeric
	_ = price.Scan(fmt.Sprintf("%f", service.Price))

	// We prepare the pointers for the optional fields (sqlc.narg)
	var namePtr *string
	if service.Name != "" {
		namePtr = &service.Name
	}

	var descPtr *string
	if service.Description != "" {
		descPtr = &service.Description
	}

	var durationPtr *int32
	if service.DurationMinutes > 0 {
		durationPtr = &service.DurationMinutes
	}

	isActivePtr := &service.IsActive

	_, err := repo.queries.UpdateStoreService(ctx, postgres.UpdateStoreServiceParams{
		ID:              serviceID,
		Name:            namePtr,
		Description:     descPtr,
		Price:           price,
		DurationMinutes: durationPtr,
		IsActive:        isActivePtr,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("service with id %s: %w", serviceID, domain.ErrNotFound)
		}
		return fmt.Errorf("updating store service %s: %w", serviceID, err)
	}

	return nil
}

// DeleteStoreService method
func (repo *StoreServiceRepository) DeleteStoreService(ctx context.Context, id string) error {
	var serviceID pgtype.UUID

	if err := serviceID.Scan(id); err != nil {
		return fmt.Errorf("invalid serviceID: %w", domain.ErrInvalidInput)
	}

	err := repo.queries.DeleteStoreService(ctx, serviceID)
	if err != nil {
		return fmt.Errorf("deleting store service: %w", err)
	}

	return nil
}
