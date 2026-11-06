package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/KevinManzano5/misto-backend/internal/domain"
)

// fakeStoreServiceRepository is a minimal StoreServiceRepository test double.
type fakeStoreServiceRepository struct {
	createStoreServiceFunc          func(ctx context.Context, service *domain.StoreService) error
	getStoreServiceByIDFunc         func(ctx context.Context, id string) (*domain.StoreService, error)
	listStoreServicesByStoreFunc    func(ctx context.Context, storeID string) ([]*domain.StoreService, error)
	listAllStoreServicesByStoreFunc func(ctx context.Context, storeID string) ([]*domain.StoreService, error)
	updateStoreServiceFunc          func(ctx context.Context, service *domain.StoreService) (*domain.StoreService, error)
	deleteStoreServiceFunc          func(ctx context.Context, id string) error
}

func (f *fakeStoreServiceRepository) CreateStoreService(ctx context.Context, service *domain.StoreService) error {
	return f.createStoreServiceFunc(ctx, service)
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
	uc := NewStoreServiceUseCase(&fakeStoreServiceRepository{}, ownedStoreRepo(), assignedCategoryRepo())

	_, err := uc.Create(context.Background(), "store-1", "someone-else", "cat-1", "Haircut", "", 30, 10)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestCreate_RejectsCategoryNotAssignedToStore(t *testing.T) {
	uc := NewStoreServiceUseCase(&fakeStoreServiceRepository{}, ownedStoreRepo(), assignedCategoryRepo())

	_, err := uc.Create(context.Background(), "store-1", "owner-1", "cat-unassigned", "Haircut", "", 30, 10)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreate_Succeeds(t *testing.T) {
	var created *domain.StoreService

	repo := &fakeStoreServiceRepository{
		createStoreServiceFunc: func(ctx context.Context, service *domain.StoreService) error {
			service.ID = "service-1"
			created = service
			return nil
		},
	}

	uc := NewStoreServiceUseCase(repo, ownedStoreRepo(), assignedCategoryRepo())

	service, err := uc.Create(context.Background(), "store-1", "owner-1", "cat-1", "Haircut", "A classic haircut", 30, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if service.ID != "service-1" {
		t.Fatalf("expected the repo-assigned ID to be returned, got %q", service.ID)
	}
	if created == nil || created.StoreID != "store-1" || created.CategoryID != "cat-1" {
		t.Fatalf("expected service persisted for store-1/cat-1, got %+v", created)
	}
}

func TestUpdate_RejectsNonOwner(t *testing.T) {
	repo := &fakeStoreServiceRepository{
		getStoreServiceByIDFunc: func(ctx context.Context, id string) (*domain.StoreService, error) {
			return &domain.StoreService{ID: id, StoreID: "store-1", CategoryID: "cat-1", Name: "Haircut", DurationMinutes: 30}, nil
		},
	}

	uc := NewStoreServiceUseCase(repo, ownedStoreRepo(), assignedCategoryRepo())

	_, err := uc.Update(context.Background(), "service-1", "someone-else", nil, nil, nil, nil, nil, nil)
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

	uc := NewStoreServiceUseCase(repo, ownedStoreRepo(), assignedCategoryRepo())

	unassigned := "cat-unassigned"
	_, err := uc.Update(context.Background(), "service-1", "owner-1", &unassigned, nil, nil, nil, nil, nil)
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

	uc := NewStoreServiceUseCase(repo, ownedStoreRepo(), assignedCategoryRepo())

	newName := "Fade Haircut"
	updated, err := uc.Update(context.Background(), "service-1", "owner-1", nil, &newName, nil, nil, nil, nil)
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

	uc := NewStoreServiceUseCase(repo, ownedStoreRepo(), assignedCategoryRepo())

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

	uc := NewStoreServiceUseCase(repo, ownedStoreRepo(), assignedCategoryRepo())

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

	uc := NewStoreServiceUseCase(repo, ownedStoreRepo(), assignedCategoryRepo())

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
