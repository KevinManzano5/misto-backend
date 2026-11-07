package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/KevinManzano5/misto-backend/internal/domain"
)

// fakeStoreServiceRepository is a minimal StoreServiceRepository test double.
type fakeStoreServiceRepository struct {
	createStoreServiceWithSchedulesFunc func(ctx context.Context, service *domain.StoreService, schedules []*domain.StoreServiceSchedule) error
	getStoreServiceByIDFunc             func(ctx context.Context, id string) (*domain.StoreService, error)
	listStoreServicesByStoreFunc        func(ctx context.Context, storeID string) ([]*domain.StoreService, error)
	listAllStoreServicesByStoreFunc     func(ctx context.Context, storeID string) ([]*domain.StoreService, error)
	updateStoreServiceFunc              func(ctx context.Context, service *domain.StoreService) (*domain.StoreService, error)
	deleteStoreServiceFunc              func(ctx context.Context, id string) error
}

func (f *fakeStoreServiceRepository) CreateStoreService(ctx context.Context, service *domain.StoreService) error {
	panic("not implemented")
}

func (f *fakeStoreServiceRepository) CreateStoreServiceWithSchedules(ctx context.Context, service *domain.StoreService, schedules []*domain.StoreServiceSchedule) error {
	return f.createStoreServiceWithSchedulesFunc(ctx, service, schedules)
}

func (f *fakeStoreServiceRepository) GetStoreServiceByID(ctx context.Context, id string) (*domain.StoreService, error) {
	return f.getStoreServiceByIDFunc(ctx, id)
}

func (f *fakeStoreServiceRepository) ListStoreServicesByStore(ctx context.Context, storeID string) ([]*domain.StoreService, error) {
	return f.listStoreServicesByStoreFunc(ctx, storeID)
}

func (f *fakeStoreServiceRepository) ListAllStoreServicesByStore(ctx context.Context, storeID string) ([]*domain.StoreService, error) {
	return f.listAllStoreServicesByStoreFunc(ctx, storeID)
}

func (f *fakeStoreServiceRepository) UpdateStoreService(ctx context.Context, service *domain.StoreService) (*domain.StoreService, error) {
	return f.updateStoreServiceFunc(ctx, service)
}

func (f *fakeStoreServiceRepository) DeleteStoreService(ctx context.Context, id string) error {
	return f.deleteStoreServiceFunc(ctx, id)
}

// fakeStoreServiceStoreRepository is a minimal StoreServiceStoreRepository test double.
type fakeStoreServiceStoreRepository struct {
	getStoreByIDFunc func(ctx context.Context, storeID string) (*domain.Store, error)
}

func (f *fakeStoreServiceStoreRepository) GetStoreByID(ctx context.Context, storeID string) (*domain.Store, error) {
	return f.getStoreByIDFunc(ctx, storeID)
}

// fakeStoreServiceCategoryRepository is a minimal StoreServiceCategoryRepository test double.
type fakeStoreServiceCategoryRepository struct {
	listCategoriesForStoreFunc func(ctx context.Context, storeID string) ([]*domain.StoreCategory, error)
}

func (f *fakeStoreServiceCategoryRepository) ListCategoriesForStore(ctx context.Context, storeID string) ([]*domain.StoreCategory, error) {
	return f.listCategoriesForStoreFunc(ctx, storeID)
}

// fakeStoreServiceScheduleRepository is a minimal StoreServiceScheduleRepository test double.
type fakeStoreServiceScheduleRepositoryForService struct {
	upsertFunc func(ctx context.Context, schedule *domain.StoreServiceSchedule) error
	listFunc   func(ctx context.Context, storeServiceID string) ([]*domain.StoreServiceSchedule, error)
}

func (f *fakeStoreServiceScheduleRepositoryForService) UpsertStoreServiceSchedule(ctx context.Context, schedule *domain.StoreServiceSchedule) error {
	return f.upsertFunc(ctx, schedule)
}

func (f *fakeStoreServiceScheduleRepositoryForService) ListStoreServiceSchedules(ctx context.Context, storeServiceID string) ([]*domain.StoreServiceSchedule, error) {
	return f.listFunc(ctx, storeServiceID)
}

// ownedStoreRepo returns a store repository fake where "owner-1" owns every store.
func ownedStoreRepo() *fakeStoreServiceStoreRepository {
	return &fakeStoreServiceStoreRepository{
		getStoreByIDFunc: func(ctx context.Context, storeID string) (*domain.Store, error) {
			return &domain.Store{ID: storeID, UserID: "owner-1"}, nil
		},
	}
}

// assignedCategoryRepo returns a category repository fake where only "cat-1" is assigned to the store.
func assignedCategoryRepo() *fakeStoreServiceCategoryRepository {
	return &fakeStoreServiceCategoryRepository{
		listCategoriesForStoreFunc: func(ctx context.Context, storeID string) ([]*domain.StoreCategory, error) {
			return []*domain.StoreCategory{{ID: "cat-1", Name: "Barbershop"}}, nil
		},
	}
}

func TestCreate_RejectsNonOwner(t *testing.T) {
	uc := NewStoreServiceUseCase(&fakeStoreServiceRepository{}, ownedStoreRepo(), assignedCategoryRepo(), &fakeStoreServiceScheduleRepositoryForService{})

	_, err := uc.Create(context.Background(), "store-1", "someone-else", "cat-1", "Haircut", "", 30, 10, 0)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestCreate_RejectsCategoryNotAssignedToStore(t *testing.T) {
	uc := NewStoreServiceUseCase(&fakeStoreServiceRepository{}, ownedStoreRepo(), assignedCategoryRepo(), &fakeStoreServiceScheduleRepositoryForService{})

	_, err := uc.Create(context.Background(), "store-1", "owner-1", "cat-unassigned", "Haircut", "", 30, 10, 0)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreate_Succeeds(t *testing.T) {
	var created *domain.StoreService
	var createdSchedules []*domain.StoreServiceSchedule

	repo := &fakeStoreServiceRepository{
		createStoreServiceWithSchedulesFunc: func(ctx context.Context, service *domain.StoreService, schedules []*domain.StoreServiceSchedule) error {
			service.ID = "service-1"
			created = service
			createdSchedules = schedules
			return nil
		},
	}

	uc := NewStoreServiceUseCase(repo, ownedStoreRepo(), assignedCategoryRepo(), &fakeStoreServiceScheduleRepositoryForService{})

	service, err := uc.Create(context.Background(), "store-1", "owner-1", "cat-1", "Haircut", "A classic haircut", 30, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if service.ID != "service-1" {
		t.Fatalf("expected the repo-assigned ID to be returned, got %q", service.ID)
	}
	if created == nil || created.StoreID != "store-1" || created.CategoryID != "cat-1" {
		t.Fatalf("expected service persisted for store-1/cat-1, got %+v", created)
	}
	if len(createdSchedules) != 7 {
		t.Fatalf("expected a default schedule seeded for all 7 days, got %d", len(createdSchedules))
	}
}

func TestUpdate_RejectsNonOwner(t *testing.T) {
	repo := &fakeStoreServiceRepository{
		getStoreServiceByIDFunc: func(ctx context.Context, id string) (*domain.StoreService, error) {
			return &domain.StoreService{ID: id, StoreID: "store-1", CategoryID: "cat-1", Name: "Haircut", DurationMinutes: 30}, nil
		},
	}

	uc := NewStoreServiceUseCase(repo, ownedStoreRepo(), assignedCategoryRepo(), &fakeStoreServiceScheduleRepositoryForService{})

	_, err := uc.Update(context.Background(), "service-1", "someone-else", nil, nil, nil, nil, nil, nil, nil)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestUpdate_RejectsCategoryNotAssignedToStore(t *testing.T) {
	repo := &fakeStoreServiceRepository{
		getStoreServiceByIDFunc: func(ctx context.Context, id string) (*domain.StoreService, error) {
			return &domain.StoreService{ID: id, StoreID: "store-1", CategoryID: "cat-1", Name: "Haircut", DurationMinutes: 30}, nil
		},
	}

	uc := NewStoreServiceUseCase(repo, ownedStoreRepo(), assignedCategoryRepo(), &fakeStoreServiceScheduleRepositoryForService{})

	unassigned := "cat-unassigned"
	_, err := uc.Update(context.Background(), "service-1", "owner-1", &unassigned, nil, nil, nil, nil, nil, nil)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdate_Succeeds(t *testing.T) {
	repo := &fakeStoreServiceRepository{
		getStoreServiceByIDFunc: func(ctx context.Context, id string) (*domain.StoreService, error) {
			return &domain.StoreService{ID: id, StoreID: "store-1", CategoryID: "cat-1", Name: "Haircut", DurationMinutes: 30}, nil
		},
		updateStoreServiceFunc: func(ctx context.Context, service *domain.StoreService) (*domain.StoreService, error) {
			return service, nil
		},
	}

	uc := NewStoreServiceUseCase(repo, ownedStoreRepo(), assignedCategoryRepo(), &fakeStoreServiceScheduleRepositoryForService{})

	newName := "Fade Haircut"
	updated, err := uc.Update(context.Background(), "service-1", "owner-1", nil, &newName, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != newName {
		t.Fatalf("expected name %q, got %q", newName, updated.Name)
	}
}

func TestDelete_RejectsNonOwner(t *testing.T) {
	repo := &fakeStoreServiceRepository{
		getStoreServiceByIDFunc: func(ctx context.Context, id string) (*domain.StoreService, error) {
			return &domain.StoreService{ID: id, StoreID: "store-1", CategoryID: "cat-1", Name: "Haircut", DurationMinutes: 30}, nil
		},
	}

	uc := NewStoreServiceUseCase(repo, ownedStoreRepo(), assignedCategoryRepo(), &fakeStoreServiceScheduleRepositoryForService{})

	err := uc.Delete(context.Background(), "service-1", "someone-else")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestDelete_Succeeds(t *testing.T) {
	var deletedID string

	repo := &fakeStoreServiceRepository{
		getStoreServiceByIDFunc: func(ctx context.Context, id string) (*domain.StoreService, error) {
			return &domain.StoreService{ID: id, StoreID: "store-1", CategoryID: "cat-1", Name: "Haircut", DurationMinutes: 30}, nil
		},
		deleteStoreServiceFunc: func(ctx context.Context, id string) error {
			deletedID = id
			return nil
		},
	}

	uc := NewStoreServiceUseCase(repo, ownedStoreRepo(), assignedCategoryRepo(), &fakeStoreServiceScheduleRepositoryForService{})

	if err := uc.Delete(context.Background(), "service-1", "owner-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedID != "service-1" {
		t.Fatalf("expected service-1 to be deleted, got %q", deletedID)
	}
}

func TestListByStore_ReturnsActiveOnly(t *testing.T) {
	var calledAll bool

	repo := &fakeStoreServiceRepository{
		listStoreServicesByStoreFunc: func(ctx context.Context, storeID string) ([]*domain.StoreService, error) {
			return []*domain.StoreService{{ID: "service-1", IsActive: true}}, nil
		},
		listAllStoreServicesByStoreFunc: func(ctx context.Context, storeID string) ([]*domain.StoreService, error) {
			calledAll = true
			return nil, nil
		},
	}

	uc := NewStoreServiceUseCase(repo, ownedStoreRepo(), assignedCategoryRepo(), &fakeStoreServiceScheduleRepositoryForService{})

	services, err := uc.ListByStore(context.Background(), "store-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(services))
	}
	if calledAll {
		t.Fatalf("expected ListByStore to call the active-only repo method, not the all-services one")
	}
}

func TestUpsertSchedule_RejectsNonOwner(t *testing.T) {
	repo := &fakeStoreServiceRepository{
		getStoreServiceByIDFunc: func(ctx context.Context, id string) (*domain.StoreService, error) {
			return &domain.StoreService{ID: id, StoreID: "store-1"}, nil
		},
	}

	uc := NewStoreServiceUseCase(repo, ownedStoreRepo(), assignedCategoryRepo(), &fakeStoreServiceScheduleRepositoryForService{})

	segments := []domain.StoreServiceScheduleSegment{{OpenTime: "08:00", CloseTime: "12:00"}}
	_, err := uc.UpsertSchedule(context.Background(), "service-1", "someone-else", 1, segments, false)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestUpsertSchedule_RejectsOverlappingSegments(t *testing.T) {
	repo := &fakeStoreServiceRepository{
		getStoreServiceByIDFunc: func(ctx context.Context, id string) (*domain.StoreService, error) {
			return &domain.StoreService{ID: id, StoreID: "store-1"}, nil
		},
	}

	uc := NewStoreServiceUseCase(repo, ownedStoreRepo(), assignedCategoryRepo(), &fakeStoreServiceScheduleRepositoryForService{})

	segments := []domain.StoreServiceScheduleSegment{
		{OpenTime: "08:00", CloseTime: "12:00"},
		{OpenTime: "11:00", CloseTime: "17:30"},
	}

	_, err := uc.UpsertSchedule(context.Background(), "service-1", "owner-1", 1, segments, false)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected %v, got %v", domain.ErrInvalidInput, err)
	}
}

func TestUpsertSchedule_AcceptsTwoSegmentsWithGap(t *testing.T) {
	repo := &fakeStoreServiceRepository{
		getStoreServiceByIDFunc: func(ctx context.Context, id string) (*domain.StoreService, error) {
			return &domain.StoreService{ID: id, StoreID: "store-1"}, nil
		},
	}

	var saved *domain.StoreServiceSchedule
	scheduleRepo := &fakeStoreServiceScheduleRepositoryForService{
		upsertFunc: func(ctx context.Context, schedule *domain.StoreServiceSchedule) error {
			saved = schedule
			return nil
		},
	}

	uc := NewStoreServiceUseCase(repo, ownedStoreRepo(), assignedCategoryRepo(), scheduleRepo)

	segments := []domain.StoreServiceScheduleSegment{
		{OpenTime: "08:00", CloseTime: "12:00"},
		{OpenTime: "13:30", CloseTime: "17:30"},
	}

	schedule, err := uc.UpsertSchedule(context.Background(), "service-1", "owner-1", 1, segments, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(schedule.Segments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(schedule.Segments))
	}
	if saved == nil {
		t.Fatalf("expected UpsertStoreServiceSchedule to be called")
	}
}

func TestGetAppUsageFeePercentage_ReturnsStoreFee(t *testing.T) {
	storeRepo := &fakeStoreServiceStoreRepository{
		getStoreByIDFunc: func(ctx context.Context, storeID string) (*domain.Store, error) {
			return &domain.Store{ID: storeID, AppUsageFeePercentage: 10}, nil
		},
	}
	uc := NewStoreServiceUseCase(&fakeStoreServiceRepository{}, storeRepo, assignedCategoryRepo(), &fakeStoreServiceScheduleRepositoryForService{})

	got, err := uc.GetAppUsageFeePercentage(context.Background(), "store-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 10 {
		t.Errorf("GetAppUsageFeePercentage() = %d, want 10", got)
	}
}

func TestGetAppUsageFeePercentage_PropagatesNotFound(t *testing.T) {
	storeRepo := &fakeStoreServiceStoreRepository{
		getStoreByIDFunc: func(ctx context.Context, storeID string) (*domain.Store, error) {
			return nil, fmt.Errorf("store with id %s: %w", storeID, domain.ErrNotFound)
		},
	}
	uc := NewStoreServiceUseCase(&fakeStoreServiceRepository{}, storeRepo, assignedCategoryRepo(), &fakeStoreServiceScheduleRepositoryForService{})

	_, err := uc.GetAppUsageFeePercentage(context.Background(), "missing-store")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("error = %v, want wrapping domain.ErrNotFound", err)
	}
}
