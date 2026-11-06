package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/KevinManzano5/misto-backend/internal/domain"
)

// fakeStoreCategoryAssignmentRepository is a minimal StoreCategoryAssignmentRepository test double.
type fakeStoreCategoryAssignmentRepository struct {
	listCategoriesForStoreFunc func(ctx context.Context, storeID string) ([]*domain.StoreCategory, error)
	replaceStoreCategoriesFunc func(ctx context.Context, storeID string, categoryIDs []string) error
}

func (f *fakeStoreCategoryAssignmentRepository) ListCategoriesForStore(ctx context.Context, storeID string) ([]*domain.StoreCategory, error) {
	return f.listCategoriesForStoreFunc(ctx, storeID)
}

func (f *fakeStoreCategoryAssignmentRepository) ReplaceStoreCategories(ctx context.Context, storeID string, categoryIDs []string) error {
	return f.replaceStoreCategoriesFunc(ctx, storeID, categoryIDs)
}

// fakeStoreCategoryAssignmentStoreRepository is a minimal StoreCategoryAssignmentStoreRepository test double.
type fakeStoreCategoryAssignmentStoreRepository struct {
	getStoreByIDFunc func(ctx context.Context, storeID string) (*domain.Store, error)
}

func (f *fakeStoreCategoryAssignmentStoreRepository) GetStoreByID(ctx context.Context, storeID string) (*domain.Store, error) {
	return f.getStoreByIDFunc(ctx, storeID)
}

func TestReplaceCategories_RejectsNonOwner(t *testing.T) {
	uc := NewStoreCategoryAssignmentUseCase(
		&fakeStoreCategoryAssignmentRepository{},
		&fakeStoreCategoryAssignmentStoreRepository{
			getStoreByIDFunc: func(ctx context.Context, storeID string) (*domain.Store, error) {
				return &domain.Store{ID: storeID, UserID: "owner-1"}, nil
			},
		},
	)

	_, err := uc.ReplaceCategories(context.Background(), "store-1", "someone-else", []string{"cat-1"})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestReplaceCategories_RejectsTooManyCategories(t *testing.T) {
	uc := NewStoreCategoryAssignmentUseCase(
		&fakeStoreCategoryAssignmentRepository{},
		&fakeStoreCategoryAssignmentStoreRepository{
			getStoreByIDFunc: func(ctx context.Context, storeID string) (*domain.Store, error) {
				return &domain.Store{ID: storeID, UserID: "owner-1"}, nil
			},
		},
	)

	categoryIDs := []string{"cat-1", "cat-2", "cat-3", "cat-4", "cat-5", "cat-6"}

	_, err := uc.ReplaceCategories(context.Background(), "store-1", "owner-1", categoryIDs)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestReplaceCategories_PropagatesUnknownCategoryID(t *testing.T) {
	repoErr := fmt.Errorf("%w: referenced category does not exist", domain.ErrInvalidInput)

	uc := NewStoreCategoryAssignmentUseCase(
		&fakeStoreCategoryAssignmentRepository{
			replaceStoreCategoriesFunc: func(ctx context.Context, storeID string, categoryIDs []string) error {
				return repoErr
			},
		},
		&fakeStoreCategoryAssignmentStoreRepository{
			getStoreByIDFunc: func(ctx context.Context, storeID string) (*domain.Store, error) {
				return &domain.Store{ID: storeID, UserID: "owner-1"}, nil
			},
		},
	)

	_, err := uc.ReplaceCategories(context.Background(), "store-1", "owner-1", []string{"unknown-cat"})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestReplaceCategories_DedupesAndSucceeds(t *testing.T) {
	var gotCategoryIDs []string

	uc := NewStoreCategoryAssignmentUseCase(
		&fakeStoreCategoryAssignmentRepository{
			replaceStoreCategoriesFunc: func(ctx context.Context, storeID string, categoryIDs []string) error {
				gotCategoryIDs = categoryIDs
				return nil
			},
			listCategoriesForStoreFunc: func(ctx context.Context, storeID string) ([]*domain.StoreCategory, error) {
				return []*domain.StoreCategory{{ID: "cat-1", Name: "Barbershop"}}, nil
			},
		},
		&fakeStoreCategoryAssignmentStoreRepository{
			getStoreByIDFunc: func(ctx context.Context, storeID string) (*domain.Store, error) {
				return &domain.Store{ID: storeID, UserID: "owner-1"}, nil
			},
		},
	)

	categories, err := uc.ReplaceCategories(context.Background(), "store-1", "owner-1", []string{"cat-1", "cat-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotCategoryIDs) != 1 {
		t.Fatalf("expected duplicates to be removed, got %v", gotCategoryIDs)
	}
	if len(categories) != 1 {
		t.Fatalf("expected 1 category in response, got %d", len(categories))
	}
}

func TestListStoreCategories_ReturnsAssigned(t *testing.T) {
	want := []*domain.StoreCategory{{ID: "cat-1", Name: "Barbershop"}}

	uc := NewStoreCategoryAssignmentUseCase(
		&fakeStoreCategoryAssignmentRepository{
			listCategoriesForStoreFunc: func(ctx context.Context, storeID string) ([]*domain.StoreCategory, error) {
				return want, nil
			},
		},
		&fakeStoreCategoryAssignmentStoreRepository{},
	)

	got, err := uc.ListStoreCategories(context.Background(), "store-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d categories, got %d", len(want), len(got))
	}
}
