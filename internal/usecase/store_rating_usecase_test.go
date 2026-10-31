package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/KevinManzano5/misto-backend/internal/domain"
)

// fakeStoreRatingRepository is a minimal StoreRatingRepository test double; only the methods
// exercised by a given test need a non-nil func field.
type fakeStoreRatingRepository struct {
	createStoreRatingFunc             func(ctx context.Context, rating *domain.StoreRating) error
	getStoreRatingByIDFunc            func(ctx context.Context, id string) (*domain.StoreRating, error)
	getStoreRatingByAppointmentIDFunc func(ctx context.Context, appointmentID string) (*domain.StoreRating, error)
	listStoreRatingsByStoreFunc       func(ctx context.Context, storeID string, limit, offset int32) ([]*domain.StoreRatingWithClient, error)
	countStoreRatingsByStoreFunc      func(ctx context.Context, storeID string) (int64, error)
	getStoreRatingSummaryByStoreFunc  func(ctx context.Context, storeID string) (*domain.StoreRatingSummary, error)
	updateStoreRatingFunc             func(ctx context.Context, id string, score int32, comment string) (*domain.StoreRating, error)
	deleteStoreRatingFunc             func(ctx context.Context, id string) error
}

func (f *fakeStoreRatingRepository) CreateStoreRating(ctx context.Context, rating *domain.StoreRating) error {
	return f.createStoreRatingFunc(ctx, rating)
}

func (f *fakeStoreRatingRepository) GetStoreRatingByID(ctx context.Context, id string) (*domain.StoreRating, error) {
	return f.getStoreRatingByIDFunc(ctx, id)
}

func (f *fakeStoreRatingRepository) GetStoreRatingByAppointmentID(ctx context.Context, appointmentID string) (*domain.StoreRating, error) {
	return f.getStoreRatingByAppointmentIDFunc(ctx, appointmentID)
}

func (f *fakeStoreRatingRepository) ListStoreRatingsByStore(ctx context.Context, storeID string, limit, offset int32) ([]*domain.StoreRatingWithClient, error) {
	return f.listStoreRatingsByStoreFunc(ctx, storeID, limit, offset)
}

func (f *fakeStoreRatingRepository) CountStoreRatingsByStore(ctx context.Context, storeID string) (int64, error) {
	return f.countStoreRatingsByStoreFunc(ctx, storeID)
}

func (f *fakeStoreRatingRepository) GetStoreRatingSummaryByStore(ctx context.Context, storeID string) (*domain.StoreRatingSummary, error) {
	return f.getStoreRatingSummaryByStoreFunc(ctx, storeID)
}

func (f *fakeStoreRatingRepository) UpdateStoreRating(ctx context.Context, id string, score int32, comment string) (*domain.StoreRating, error) {
	return f.updateStoreRatingFunc(ctx, id, score, comment)
}

func (f *fakeStoreRatingRepository) DeleteStoreRating(ctx context.Context, id string) error {
	return f.deleteStoreRatingFunc(ctx, id)
}

// fakeStoreRatingAppointmentRepository is a minimal StoreRatingAppointmentRepository test double.
type fakeStoreRatingAppointmentRepository struct {
	getAppointmentByIDFunc func(ctx context.Context, id string) (*domain.Appointment, error)
}

func (f *fakeStoreRatingAppointmentRepository) GetAppointmentByID(ctx context.Context, id string) (*domain.Appointment, error) {
	return f.getAppointmentByIDFunc(ctx, id)
}

func TestCreate_RejectsInvalidScore(t *testing.T) {
	uc := NewStoreRatingUseCase(&fakeStoreRatingRepository{}, &fakeStoreRatingAppointmentRepository{})

	for _, score := range []int32{0, 6} {
		_, err := uc.Create(context.Background(), "appt-1", "user-1", score, "")
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("score %d: expected ErrInvalidInput, got %v", score, err)
		}
	}
}

func TestCreate(t *testing.T) {
	baseAppointment := &domain.Appointment{
		ID:      "appt-1",
		StoreID: "store-1",
		UserID:  "client-1",
		Status:  domain.AppointmentStatusCompleted,
	}

	tests := []struct {
		name             string
		requesterUserID  string
		appointment      *domain.Appointment
		getByAppointment func(ctx context.Context, appointmentID string) (*domain.StoreRating, error)
		createFunc       func(ctx context.Context, rating *domain.StoreRating) error
		wantErr          error
	}{
		{
			name:            "client of a completed, unrated appointment succeeds",
			requesterUserID: "client-1",
			appointment:     baseAppointment,
			getByAppointment: func(ctx context.Context, appointmentID string) (*domain.StoreRating, error) {
				return nil, fmt.Errorf("not rated yet: %w", domain.ErrNotFound)
			},
			createFunc: func(ctx context.Context, rating *domain.StoreRating) error {
				return nil
			},
		},
		{
			name:            "a stranger cannot rate the appointment",
			requesterUserID: "someone-else",
			appointment:     baseAppointment,
			wantErr:         domain.ErrForbidden,
		},
		{
			name:            "a pending appointment cannot be rated",
			requesterUserID: "client-1",
			appointment: &domain.Appointment{
				ID:      "appt-1",
				StoreID: "store-1",
				UserID:  "client-1",
				Status:  domain.AppointmentStatusPending,
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name:            "an already-rated appointment is rejected",
			requesterUserID: "client-1",
			appointment:     baseAppointment,
			getByAppointment: func(ctx context.Context, appointmentID string) (*domain.StoreRating, error) {
				return &domain.StoreRating{ID: "rating-1"}, nil
			},
			wantErr: domain.ErrAlreadyExists,
		},
		{
			name:            "a race detected only at the DB layer is propagated",
			requesterUserID: "client-1",
			appointment:     baseAppointment,
			getByAppointment: func(ctx context.Context, appointmentID string) (*domain.StoreRating, error) {
				return nil, fmt.Errorf("not rated yet: %w", domain.ErrNotFound)
			},
			createFunc: func(ctx context.Context, rating *domain.StoreRating) error {
				return fmt.Errorf("db race: %w", domain.ErrAlreadyExists)
			},
			wantErr: domain.ErrAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeStoreRatingRepository{
				getStoreRatingByAppointmentIDFunc: tt.getByAppointment,
				createStoreRatingFunc:             tt.createFunc,
			}
			appointmentRepo := &fakeStoreRatingAppointmentRepository{
				getAppointmentByIDFunc: func(ctx context.Context, id string) (*domain.Appointment, error) {
					return tt.appointment, nil
				},
			}
			uc := NewStoreRatingUseCase(repo, appointmentRepo)

			_, err := uc.Create(context.Background(), tt.appointment.ID, tt.requesterUserID, 5, "great service")

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestUpdate_RejectsNonAuthor(t *testing.T) {
	repo := &fakeStoreRatingRepository{
		getStoreRatingByIDFunc: func(ctx context.Context, id string) (*domain.StoreRating, error) {
			return &domain.StoreRating{ID: id, UserID: "author-1"}, nil
		},
	}
	uc := NewStoreRatingUseCase(repo, &fakeStoreRatingAppointmentRepository{})

	_, err := uc.Update(context.Background(), "rating-1", "someone-else", 4, "")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestUpdate_AuthorSucceeds(t *testing.T) {
	repo := &fakeStoreRatingRepository{
		getStoreRatingByIDFunc: func(ctx context.Context, id string) (*domain.StoreRating, error) {
			return &domain.StoreRating{ID: id, UserID: "author-1"}, nil
		},
		updateStoreRatingFunc: func(ctx context.Context, id string, score int32, comment string) (*domain.StoreRating, error) {
			return &domain.StoreRating{ID: id, UserID: "author-1", Score: score, Comment: comment}, nil
		},
	}
	uc := NewStoreRatingUseCase(repo, &fakeStoreRatingAppointmentRepository{})

	updated, err := uc.Update(context.Background(), "rating-1", "author-1", 4, "updated comment")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Score != 4 || updated.Comment != "updated comment" {
		t.Fatalf("unexpected updated rating: %+v", updated)
	}
}

func TestDelete_RejectsNonAuthor(t *testing.T) {
	repo := &fakeStoreRatingRepository{
		getStoreRatingByIDFunc: func(ctx context.Context, id string) (*domain.StoreRating, error) {
			return &domain.StoreRating{ID: id, UserID: "author-1"}, nil
		},
	}
	uc := NewStoreRatingUseCase(repo, &fakeStoreRatingAppointmentRepository{})

	err := uc.Delete(context.Background(), "rating-1", "someone-else")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestDelete_AuthorSucceeds(t *testing.T) {
	deleted := false
	repo := &fakeStoreRatingRepository{
		getStoreRatingByIDFunc: func(ctx context.Context, id string) (*domain.StoreRating, error) {
			return &domain.StoreRating{ID: id, UserID: "author-1"}, nil
		},
		deleteStoreRatingFunc: func(ctx context.Context, id string) error {
			deleted = true
			return nil
		},
	}
	uc := NewStoreRatingUseCase(repo, &fakeStoreRatingAppointmentRepository{})

	if err := uc.Delete(context.Background(), "rating-1", "author-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Fatalf("expected repository Delete to be called")
	}
}

func TestListByStore_ReturnsFromRepository(t *testing.T) {
	want := []*domain.StoreRatingWithClient{
		{StoreRating: domain.StoreRating{ID: "rating-1"}, ClientUsername: "alice"},
	}
	repo := &fakeStoreRatingRepository{
		listStoreRatingsByStoreFunc: func(ctx context.Context, storeID string, limit, offset int32) ([]*domain.StoreRatingWithClient, error) {
			if limit != 20 || offset != 0 {
				t.Fatalf("expected default page/pageSize (limit=20, offset=0), got limit=%d offset=%d", limit, offset)
			}
			return want, nil
		},
		countStoreRatingsByStoreFunc: func(ctx context.Context, storeID string) (int64, error) {
			return 1, nil
		},
	}
	uc := NewStoreRatingUseCase(repo, &fakeStoreRatingAppointmentRepository{})

	got, total, err := uc.ListByStore(context.Background(), "store-1", 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 || len(got) != 1 || got[0].ID != "rating-1" {
		t.Fatalf("unexpected result: got=%+v total=%d", got, total)
	}
}

func TestGetSummaryByStore_ReturnsFromRepository(t *testing.T) {
	want := &domain.StoreRatingSummary{StoreID: "store-1", AverageScore: 4.5, RatingCount: 2}
	repo := &fakeStoreRatingRepository{
		getStoreRatingSummaryByStoreFunc: func(ctx context.Context, storeID string) (*domain.StoreRatingSummary, error) {
			return want, nil
		},
	}
	uc := NewStoreRatingUseCase(repo, &fakeStoreRatingAppointmentRepository{})

	got, err := uc.GetSummaryByStore(context.Background(), "store-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.AverageScore != 4.5 || got.RatingCount != 2 {
		t.Fatalf("unexpected summary: %+v", got)
	}
}
