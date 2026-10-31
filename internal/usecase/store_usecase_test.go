package usecase

import (
	"context"
	"testing"

	"github.com/KevinManzano5/misto-backend/internal/domain"
)

// fakeStoreRepository is a minimal StoreRepository test double; only the methods exercised by a
// given test need a non-nil func field.
type fakeStoreRepository struct {
	createStoreWithSchedulesFunc func(ctx context.Context, store *domain.Store, schedules []*domain.StoreSchedule) error
	getStoreByIDFunc             func(ctx context.Context, id string) (*domain.Store, error)
	updateStoreFunc              func(ctx context.Context, store *domain.Store) error
	listStoresWithoutSlugFunc    func(ctx context.Context) ([]*domain.Store, error)
}

func (f *fakeStoreRepository) CreateStore(ctx context.Context, store *domain.Store) error {
	panic("not implemented")
}

func (f *fakeStoreRepository) CreateStoreWithSchedules(ctx context.Context, store *domain.Store, schedules []*domain.StoreSchedule) error {
	return f.createStoreWithSchedulesFunc(ctx, store, schedules)
}

func (f *fakeStoreRepository) GetStoreByID(ctx context.Context, storeID string) (*domain.Store, error) {
	return f.getStoreByIDFunc(ctx, storeID)
}

func (f *fakeStoreRepository) GetStoreByUserID(ctx context.Context, userID string) (*domain.Store, error) {
	panic("not implemented")
}

func (f *fakeStoreRepository) UpdateStore(ctx context.Context, store *domain.Store) error {
	return f.updateStoreFunc(ctx, store)
}

func (f *fakeStoreRepository) DeleteStore(ctx context.Context, id string) error {
	panic("not implemented")
}

func (f *fakeStoreRepository) ListStores(ctx context.Context) ([]*domain.Store, error) {
	panic("not implemented")
}

func (f *fakeStoreRepository) ListStoresWithoutSlug(ctx context.Context) ([]*domain.Store, error) {
	return f.listStoresWithoutSlugFunc(ctx)
}

// fakeScheduleRepository is a minimal ScheduleRepository test double.
type fakeScheduleRepository struct{}

func (f *fakeScheduleRepository) UpsertStoreSchedule(ctx context.Context, schedule *domain.StoreSchedule) error {
	panic("not implemented")
}

func (f *fakeScheduleRepository) ListStoreSchedules(ctx context.Context, storeID string) ([]*domain.StoreSchedule, error) {
	panic("not implemented")
}

func TestCreate_SetsSlugFromName(t *testing.T) {
	var captured *domain.Store
	repo := &fakeStoreRepository{
		createStoreWithSchedulesFunc: func(ctx context.Context, store *domain.Store, schedules []*domain.StoreSchedule) error {
			captured = store
			return nil
		},
	}
	uc := NewStoreUseCase(repo, &fakeScheduleRepository{})

	_, err := uc.Create(context.Background(), "user-1", "Café Sol", "desc", true, 30, "America/Bogota", true, "", "", true, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := domain.SlugFromName("Café Sol"); captured.Slug != want {
		t.Errorf("Slug = %q, want %q", captured.Slug, want)
	}
}

func TestUpdate_NameChangeRecomputesSlug(t *testing.T) {
	existing := &domain.Store{ID: "store-1", UserID: "user-1", Name: "Old Name", Slug: "old-name", AppointmentDuration: 30, Timezone: "America/Bogota"}

	var captured *domain.Store
	repo := &fakeStoreRepository{
		getStoreByIDFunc: func(ctx context.Context, id string) (*domain.Store, error) {
			return existing, nil
		},
		updateStoreFunc: func(ctx context.Context, store *domain.Store) error {
			captured = store
			return nil
		},
	}
	uc := NewStoreUseCase(repo, &fakeScheduleRepository{})

	newName := "New Name"
	_, err := uc.Update(context.Background(), "store-1", &newName, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := domain.SlugFromName(newName); captured.Slug != want {
		t.Errorf("Slug = %q, want %q", captured.Slug, want)
	}
}

func TestUpdate_NoNameChangeKeepsSlug(t *testing.T) {
	existing := &domain.Store{ID: "store-1", UserID: "user-1", Name: "Same Name", Slug: "same-name", AppointmentDuration: 30, Timezone: "America/Bogota"}

	var captured *domain.Store
	repo := &fakeStoreRepository{
		getStoreByIDFunc: func(ctx context.Context, id string) (*domain.Store, error) {
			return existing, nil
		},
		updateStoreFunc: func(ctx context.Context, store *domain.Store) error {
			captured = store
			return nil
		},
	}
	uc := NewStoreUseCase(repo, &fakeScheduleRepository{})

	newDescription := "updated description"
	_, err := uc.Update(context.Background(), "store-1", nil, &newDescription, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured.Slug != "same-name" {
		t.Errorf("expected Slug to remain %q, got %q", "same-name", captured.Slug)
	}
}

func TestBackfillMissingSlugs(t *testing.T) {
	missing := []*domain.Store{
		{ID: "store-1", Name: "Café Sol"},
		{ID: "store-2", Name: "Panadería El Sol"},
	}

	var updated []*domain.Store
	repo := &fakeStoreRepository{
		listStoresWithoutSlugFunc: func(ctx context.Context) ([]*domain.Store, error) {
			return missing, nil
		},
		updateStoreFunc: func(ctx context.Context, store *domain.Store) error {
			updated = append(updated, store)
			return nil
		},
	}
	uc := NewStoreUseCase(repo, &fakeScheduleRepository{})

	count, err := uc.BackfillMissingSlugs(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 stores updated, got %d", count)
	}
	if len(updated) != 2 {
		t.Fatalf("expected UpdateStore called twice, got %d", len(updated))
	}
	if updated[0].Slug != domain.SlugFromName("Café Sol") {
		t.Errorf("Slug = %q, want %q", updated[0].Slug, domain.SlugFromName("Café Sol"))
	}
	if updated[1].Slug != domain.SlugFromName("Panadería El Sol") {
		t.Errorf("Slug = %q, want %q", updated[1].Slug, domain.SlugFromName("Panadería El Sol"))
	}
}
