package usecase

import (
	"context"
	"testing"

	"github.com/KevinManzano5/misto-backend/internal/domain"
)

// fakeStoreCategoryRepository is a minimal StoreCategoryRepository test double.
type fakeStoreCategoryRepository struct {
	listStoreCategoriesFunc func(ctx context.Context) ([]*domain.StoreCategory, error)
}

func (f *fakeStoreCategoryRepository) ListStoreCategories(ctx context.Context) ([]*domain.StoreCategory, error) {
	return f.listStoreCategoriesFunc(ctx)
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
