package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/KevinManzano5/misto-backend/internal/domain"
)

// defaultAppointmentDuration is applied to a store when it is created without one.
const defaultAppointmentDuration = 30

// defaultTimezone is applied to a store when it is created without one.
const defaultTimezone = "America/Bogota"

// validateTimezone checks that name is a loadable IANA timezone (e.g. "America/Bogota").
func validateTimezone(name string) error {
	if _, err := time.LoadLocation(name); err != nil {
		return fmt.Errorf("%w: invalid timezone %q", domain.ErrInvalidInput, name)
	}

	return nil
}

// StoreRepository interface
type StoreRepository interface {
	CreateStore(ctx context.Context, store *domain.Store) error
	GetStoreByID(ctx context.Context, storeID string) (*domain.Store, error)
	GetStoreByUserID(ctx context.Context, userID string) (*domain.Store, error)
	GetStoreBySlug(ctx context.Context, slug string) (*domain.Store, error)
	UpdateStore(ctx context.Context, store *domain.Store) error
	DeleteStore(ctx context.Context, id string) error
	ListStores(ctx context.Context) ([]*domain.Store, error)
	ListStoresWithoutSlug(ctx context.Context) ([]*domain.Store, error)
}

// StoreRatingSummaryRepository interface (narrowed to what StoreUseCase needs to embed a store's
// rating average/count into its own responses).
type StoreRatingSummaryRepository interface {
	GetStoreRatingSummaryByStore(ctx context.Context, storeID string) (*domain.StoreRatingSummary, error)
	GetStoreRatingSummariesByStores(ctx context.Context, storeIDs []string) ([]*domain.StoreRatingSummary, error)
}

// StoreUseCase struct
type StoreUseCase struct {
	repo       StoreRepository
	ratingRepo StoreRatingSummaryRepository
}

// NewStoreUseCase constructor
func NewStoreUseCase(repo StoreRepository, ratingRepo StoreRatingSummaryRepository) *StoreUseCase {
	return &StoreUseCase{
		repo:       repo,
		ratingRepo: ratingRepo,
	}
}

// zeroStoreRatingSummary is returned for a store with no ratings yet, since the repository
// simply omits such stores from its results rather than returning a zeroed row.
func zeroStoreRatingSummary(storeID string) *domain.StoreRatingSummary {
	return &domain.StoreRatingSummary{StoreID: storeID}
}

// GetRatingSummary returns the average score and rating count for a single store. Public: no
// authorization check.
func (usecase *StoreUseCase) GetRatingSummary(ctx context.Context, storeID string) (*domain.StoreRatingSummary, error) {
	summary, err := usecase.ratingRepo.GetStoreRatingSummaryByStore(ctx, storeID)
	if err != nil {
		return nil, fmt.Errorf("getting store rating summary: %w", err)
	}

	return summary, nil
}

// ListRatingSummaries returns the average score and rating count for each of storeIDs, keyed by
// store ID, computed in a single query to avoid N+1 lookups when building a store listing. A
// storeID with no ratings yet is included with a zero summary.
func (usecase *StoreUseCase) ListRatingSummaries(ctx context.Context, storeIDs []string) (map[string]*domain.StoreRatingSummary, error) {
	summaries := make(map[string]*domain.StoreRatingSummary, len(storeIDs))
	for _, storeID := range storeIDs {
		summaries[storeID] = zeroStoreRatingSummary(storeID)
	}

	if len(storeIDs) == 0 {
		return summaries, nil
	}

	results, err := usecase.ratingRepo.GetStoreRatingSummariesByStores(ctx, storeIDs)
	if err != nil {
		return nil, fmt.Errorf("listing store rating summaries: %w", err)
	}

	for _, result := range results {
		summaries[result.StoreID] = result
	}

	return summaries, nil
}

// Create method creates a store
func (usecase *StoreUseCase) Create(ctx context.Context, userID, name, description string, isActive bool, appointmentDuration int32, timezone string, autoConfirmAppointments bool, email, address string, allowSameDayAppointments bool, phoneNumber string) (*domain.Store, error) {
	if appointmentDuration <= 0 {
		appointmentDuration = defaultAppointmentDuration
	}

	if timezone == "" {
		timezone = defaultTimezone
	} else if err := validateTimezone(timezone); err != nil {
		return nil, err
	}

	slug := domain.SlugFromName(name)
	store := domain.NewStore(userID, name, slug, description, isActive, appointmentDuration, timezone, autoConfirmAppointments, email, address, allowSameDayAppointments, phoneNumber)

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

// GetBySlug method
func (usecase *StoreUseCase) GetBySlug(ctx context.Context, slug string) (*domain.Store, error) {
	store, err := usecase.repo.GetStoreBySlug(ctx, slug)

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

// List method
func (usecase *StoreUseCase) List(ctx context.Context) ([]*domain.Store, error) {
	stores, err := usecase.repo.ListStores(ctx)

	if err != nil {
		return nil, fmt.Errorf("listing stores: %w", err)
	}

	return stores, nil
}

// Update method applies the supplied changes to a store and returns the persisted store.
func (usecase *StoreUseCase) Update(ctx context.Context, storeID string, name, description *string, isActive *bool, appointmentDuration *int32, timezone *string, autoConfirmAppointments *bool, email, address *string, allowSameDayAppointments *bool, phoneNumber *string) (*domain.Store, error) {
	store, err := usecase.repo.GetStoreByID(ctx, storeID)
	if err != nil {
		return nil, fmt.Errorf("getting store to update: %w", err)
	}

	if name != nil {
		store.Name = *name
		store.Slug = domain.SlugFromName(*name)
	}
	if description != nil {
		store.Description = *description
	}
	if isActive != nil {
		store.IsActive = *isActive
	}
	if appointmentDuration != nil {
		store.AppointmentDuration = *appointmentDuration
	}
	if timezone != nil {
		if err := validateTimezone(*timezone); err != nil {
			return nil, err
		}
		store.Timezone = *timezone
	}
	if autoConfirmAppointments != nil {
		store.AutoConfirmAppointments = *autoConfirmAppointments
	}
	if email != nil {
		store.Email = *email
	}
	if address != nil {
		store.Address = *address
	}
	if allowSameDayAppointments != nil {
		store.AllowSameDayAppointments = *allowSameDayAppointments
	}
	if phoneNumber != nil {
		store.PhoneNumber = *phoneNumber
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

// BackfillMissingSlugs computes and persists a slug for every store that doesn't have one yet
// (created before the slug column existed). It is a system/maintenance action with no requester
// to authorize against, meant to be run once via a standalone command (see
// cmd/backfillstoreslugs) rather than from an HTTP handler. Returns how many stores were updated.
func (usecase *StoreUseCase) BackfillMissingSlugs(ctx context.Context) (int, error) {
	stores, err := usecase.repo.ListStoresWithoutSlug(ctx)
	if err != nil {
		return 0, fmt.Errorf("listing stores without slug: %w", err)
	}

	for _, store := range stores {
		store.Slug = domain.SlugFromName(store.Name)
		if err := usecase.repo.UpdateStore(ctx, store); err != nil {
			return 0, fmt.Errorf("updating slug for store %s: %w", store.ID, err)
		}
	}

	return len(stores), nil
}

// Delete method
func (usecase *StoreUseCase) Delete(ctx context.Context, storeID string) error {
	err := usecase.repo.DeleteStore(ctx, storeID)

	if err != nil {
		return fmt.Errorf("deleting store: %w", err)
	}

	return nil
}
