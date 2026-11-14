package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/KevinManzano5/misto-backend/internal/domain"
)

// fakeStoreCategoryRepository is a minimal StoreCategoryRepository test double.
type fakeStoreCategoryRepository struct {
	createStoreCategoryFunc func(ctx context.Context, category *domain.StoreCategory) error
	getStoreCategoryFunc    func(ctx context.Context, id string) (*domain.StoreCategory, error)
	listStoreCategoriesFunc func(ctx context.Context) ([]*domain.StoreCategory, error)
	updateStoreCategoryFunc func(ctx context.Context, category *domain.StoreCategory) (*domain.StoreCategory, error)
	deleteStoreCategoryFunc func(ctx context.Context, id string) error
}

func (f *fakeStoreCategoryRepository) CreateStoreCategory(ctx context.Context, category *domain.StoreCategory) error {
	if f.createStoreCategoryFunc == nil {
		return errors.New("createStoreCategoryFunc not set")
	}
	return f.createStoreCategoryFunc(ctx, category)
}

func (f *fakeStoreCategoryRepository) GetStoreCategoryByID(ctx context.Context, id string) (*domain.StoreCategory, error) {
	if f.getStoreCategoryFunc == nil {
		return nil, errors.New("getStoreCategoryFunc not set")
	}
	return f.getStoreCategoryFunc(ctx, id)
}

func (f *fakeStoreCategoryRepository) ListStoreCategories(ctx context.Context) ([]*domain.StoreCategory, error) {
	return f.listStoreCategoriesFunc(ctx)
}

func (f *fakeStoreCategoryRepository) UpdateStoreCategory(ctx context.Context, category *domain.StoreCategory) (*domain.StoreCategory, error) {
	if f.updateStoreCategoryFunc == nil {
		return nil, errors.New("updateStoreCategoryFunc not set")
	}
	return f.updateStoreCategoryFunc(ctx, category)
}

func (f *fakeStoreCategoryRepository) DeleteStoreCategory(ctx context.Context, id string) error {
	if f.deleteStoreCategoryFunc == nil {
		return errors.New("deleteStoreCategoryFunc not set")
	}
	return f.deleteStoreCategoryFunc(ctx, id)
}

func TestListCategories_ReturnsCatalog(t *testing.T) {
	want := []*domain.StoreCategory{
		{ID: "cat-1", Name: "Barbershop", Slug: "barbershop"},
		{ID: "cat-2", Name: "Spa", Slug: "spa"},
	}

	uc := NewStoreCategoryUseCase(&fakeStoreCategoryRepository{
		listStoreCategoriesFunc: func(ctx context.Context) ([]*domain.StoreCategory, error) {
			return want, nil
		},
	})

	got, err := uc.ListCategories(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d categories, got %d", len(want), len(got))
	}
}

func TestCreate_RejectsEmptyName(t *testing.T) {
	uc := NewStoreCategoryUseCase(&fakeStoreCategoryRepository{})

	_, err := uc.Create(context.Background(), "", "a-slug", "")
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreate_PersistsValidCategory(t *testing.T) {
	var created *domain.StoreCategory

	uc := NewStoreCategoryUseCase(&fakeStoreCategoryRepository{
		createStoreCategoryFunc: func(ctx context.Context, category *domain.StoreCategory) error {
			category.ID = "cat-1"
			created = category
			return nil
		},
	})

	got, err := uc.Create(context.Background(), "Barbershop", "barbershop", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "cat-1" || created != got {
		t.Fatalf("expected the created category to be returned, got %+v", got)
	}
}

func TestUpdate_AppliesPartialChanges(t *testing.T) {
	existing := &domain.StoreCategory{ID: "cat-1", Name: "Barbershop", Slug: "barbershop"}
	newName := "Hair Salon"

	uc := NewStoreCategoryUseCase(&fakeStoreCategoryRepository{
		getStoreCategoryFunc: func(ctx context.Context, id string) (*domain.StoreCategory, error) {
			return existing, nil
		},
		updateStoreCategoryFunc: func(ctx context.Context, category *domain.StoreCategory) (*domain.StoreCategory, error) {
			return category, nil
		},
	})

	got, err := uc.Update(context.Background(), "cat-1", &newName, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != newName || got.Slug != "barbershop" {
		t.Fatalf("expected only name to change, got %+v", got)
	}
}

func TestDelete_PropagatesRepositoryError(t *testing.T) {
	existing := &domain.StoreCategory{ID: "cat-1", Name: "Barbershop", Slug: "barbershop"}

	uc := NewStoreCategoryUseCase(&fakeStoreCategoryRepository{
		getStoreCategoryFunc: func(ctx context.Context, id string) (*domain.StoreCategory, error) {
			return existing, nil
		},
		deleteStoreCategoryFunc: func(ctx context.Context, id string) error {
			return fmt.Errorf("%w: category is in use", domain.ErrAlreadyExists)
		},
	})

	err := uc.Delete(context.Background(), "cat-1")
	if !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}
