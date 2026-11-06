package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/KevinManzano5/misto-backend/internal/domain"
)

// StoreRatingRepository interface
type StoreRatingRepository interface {
	CreateStoreRating(ctx context.Context, rating *domain.StoreRating) error
	GetStoreRatingByID(ctx context.Context, id string) (*domain.StoreRating, error)
	GetStoreRatingByAppointmentID(ctx context.Context, appointmentID string) (*domain.StoreRating, error)
	ListStoreRatingsByStore(ctx context.Context, storeID string, limit, offset int32) ([]*domain.StoreRatingWithClient, error)
	CountStoreRatingsByStore(ctx context.Context, storeID string) (int64, error)
	ListRatedAppointmentIDs(ctx context.Context, appointmentIDs []string) ([]string, error)
	UpdateStoreRating(ctx context.Context, id string, score int32, comment string) (*domain.StoreRating, error)
	DeleteStoreRating(ctx context.Context, id string) error
}

// StoreRatingAppointmentRepository interface (narrowed to what StoreRatingUseCase needs from an appointment)
type StoreRatingAppointmentRepository interface {
	GetAppointmentByID(ctx context.Context, id string) (*domain.Appointment, error)
}

// StoreRatingUseCase struct
type StoreRatingUseCase struct {
	repo            StoreRatingRepository
	appointmentRepo StoreRatingAppointmentRepository
}

const (
	defaultRatingsPage     = 1
	defaultRatingsPageSize = 20
	maxRatingsPageSize     = 100
)

// NewStoreRatingUseCase constructor
func NewStoreRatingUseCase(repo StoreRatingRepository, appointmentRepo StoreRatingAppointmentRepository) *StoreRatingUseCase {
	return &StoreRatingUseCase{
		repo:            repo,
		appointmentRepo: appointmentRepo,
	}
}

// Create leaves a rating for a completed appointment. Only the client who booked the appointment
// (requesterUserID matches the appointment's UserID) may rate it, only once it is completed, and
// only once per appointment: a second attempt is rejected with domain.ErrAlreadyExists.
func (usecase *StoreRatingUseCase) Create(ctx context.Context, appointmentID, requesterUserID string, score int32, comment string) (*domain.StoreRating, error) {
	if score < 1 || score > 5 {
		return nil, fmt.Errorf("%w: score must be between 1 and 5", domain.ErrInvalidInput)
	}

	appointment, err := usecase.appointmentRepo.GetAppointmentByID(ctx, appointmentID)
	if err != nil {
		return nil, fmt.Errorf("getting appointment: %w", err)
	}

	if appointment.UserID != requesterUserID {
		return nil, fmt.Errorf("%w: only the client who booked this appointment can rate it", domain.ErrForbidden)
	}

	if appointment.Status != domain.AppointmentStatusCompleted {
		return nil, fmt.Errorf("%w: only a completed appointment can be rated", domain.ErrInvalidInput)
	}

	_, err = usecase.repo.GetStoreRatingByAppointmentID(ctx, appointmentID)
	if err == nil {
		return nil, fmt.Errorf("%w: this appointment has already been rated", domain.ErrAlreadyExists)
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("checking existing rating: %w", err)
	}

	rating := domain.NewStoreRating(appointmentID, appointment.StoreID, requesterUserID, score, comment)

	if err := usecase.repo.CreateStoreRating(ctx, rating); err != nil {
		return nil, fmt.Errorf("creating store rating: %w", err)
	}

	return rating, nil
}

// ListByStore returns a page of a store's ratings, newest first, each paired with the basic
// profile of the client who left it, along with the total count across all pages. Public: no
// authorization check. page/pageSize are normalized with sane defaults and an upper bound.
func (usecase *StoreRatingUseCase) ListByStore(ctx context.Context, storeID string, page, pageSize int32) ([]*domain.StoreRatingWithClient, int64, error) {
	if page < 1 {
		page = defaultRatingsPage
	}
	if pageSize < 1 {
		pageSize = defaultRatingsPageSize
	}
	if pageSize > maxRatingsPageSize {
		pageSize = maxRatingsPageSize
	}

	offset := (page - 1) * pageSize

	ratings, err := usecase.repo.ListStoreRatingsByStore(ctx, storeID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("listing store ratings: %w", err)
	}

	total, err := usecase.repo.CountStoreRatingsByStore(ctx, storeID)
	if err != nil {
		return nil, 0, fmt.Errorf("counting store ratings: %w", err)
	}

	return ratings, total, nil
}

// ListRatedAppointmentIDs returns which of appointmentIDs already has a rating, as a set keyed
// by appointment ID. An appointmentID with no rating is simply absent from the map. Used to
// annotate a client's own appointment list with whether each one can still be rated.
func (usecase *StoreRatingUseCase) ListRatedAppointmentIDs(ctx context.Context, appointmentIDs []string) (map[string]bool, error) {
	rated := make(map[string]bool, len(appointmentIDs))

	if len(appointmentIDs) == 0 {
		return rated, nil
	}

	results, err := usecase.repo.ListRatedAppointmentIDs(ctx, appointmentIDs)
	if err != nil {
		return nil, fmt.Errorf("listing rated appointment IDs: %w", err)
	}

	for _, appointmentID := range results {
		rated[appointmentID] = true
	}

	return rated, nil
}

// Update edits the score/comment of a rating. Only the rating's author may edit it.
func (usecase *StoreRatingUseCase) Update(ctx context.Context, ratingID, requesterUserID string, score int32, comment string) (*domain.StoreRating, error) {
	if score < 1 || score > 5 {
		return nil, fmt.Errorf("%w: score must be between 1 and 5", domain.ErrInvalidInput)
	}

	rating, err := usecase.repo.GetStoreRatingByID(ctx, ratingID)
	if err != nil {
		return nil, fmt.Errorf("getting store rating: %w", err)
	}

	if rating.UserID != requesterUserID {
		return nil, fmt.Errorf("%w: only the author of this rating can edit it", domain.ErrForbidden)
	}

	updated, err := usecase.repo.UpdateStoreRating(ctx, ratingID, score, comment)
	if err != nil {
		return nil, fmt.Errorf("updating store rating: %w", err)
	}

	return updated, nil
}

// Delete removes a rating. Only the rating's author may delete it.
func (usecase *StoreRatingUseCase) Delete(ctx context.Context, ratingID, requesterUserID string) error {
	rating, err := usecase.repo.GetStoreRatingByID(ctx, ratingID)
	if err != nil {
		return fmt.Errorf("getting store rating: %w", err)
	}

	if rating.UserID != requesterUserID {
		return fmt.Errorf("%w: only the author of this rating can delete it", domain.ErrForbidden)
	}

	if err := usecase.repo.DeleteStoreRating(ctx, ratingID); err != nil {
		return fmt.Errorf("deleting store rating: %w", err)
	}

	return nil
}
