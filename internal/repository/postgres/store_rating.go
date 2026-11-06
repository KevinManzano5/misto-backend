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

// StoreRatingRepository struct
type StoreRatingRepository struct {
	pool    *pgxpool.Pool
	queries *postgres.Queries
}

// NewStoreRatingRepository function
func NewStoreRatingRepository(pool *pgxpool.Pool) *StoreRatingRepository {
	return &StoreRatingRepository{
		pool:    pool,
		queries: postgres.New(pool),
	}
}

// toDomainStoreRating function
func toDomainStoreRating(rating postgres.StoreRating) *domain.StoreRating {
	var comment string
	if rating.Comment != nil {
		comment = *rating.Comment
	}

	return &domain.StoreRating{
		ID:            rating.ID.String(),
		AppointmentID: rating.AppointmentID.String(),
		StoreID:       rating.StoreID.String(),
		UserID:        rating.ClientID.String(),
		Score:         int32(rating.Score),
		Comment:       comment,
		CreatedAt:     rating.CreatedAt.Time.UTC(),
		UpdatedAt:     rating.UpdatedAt.Time.UTC(),
	}
}

// translateStoreRatingWriteError maps known Postgres constraint violations from writing a store
// rating to the shared domain sentinel errors.
func translateStoreRatingWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503": // foreign_key_violation: appointment_id, store_id, or client_id does not reference an existing row
			return fmt.Errorf("%w: referenced appointment, store, or client does not exist", domain.ErrInvalidInput)
		case "23505": // unique_violation: appointment already has a rating
			return fmt.Errorf("%w: this appointment has already been rated", domain.ErrAlreadyExists)
		}
	}

	return fmt.Errorf("writing store rating: %w", err)
}

// CreateStoreRating method
func (repo *StoreRatingRepository) CreateStoreRating(ctx context.Context, rating *domain.StoreRating) error {
	var appointmentID, storeID, clientID pgtype.UUID

	if err := appointmentID.Scan(rating.AppointmentID); err != nil {
		return fmt.Errorf("invalid appointmentID: %w", domain.ErrInvalidInput)
	}
	if err := storeID.Scan(rating.StoreID); err != nil {
		return fmt.Errorf("invalid storeID: %w", domain.ErrInvalidInput)
	}
	if err := clientID.Scan(rating.UserID); err != nil {
		return fmt.Errorf("invalid userID: %w", domain.ErrInvalidInput)
	}

	var comment *string
	if rating.Comment != "" {
		comment = &rating.Comment
	}

	result, err := repo.queries.CreateStoreRating(ctx, postgres.CreateStoreRatingParams{
		AppointmentID: appointmentID,
		StoreID:       storeID,
		ClientID:      clientID,
		Score:         int16(rating.Score),
		Comment:       comment,
	})
	if err != nil {
		return translateStoreRatingWriteError(err)
	}

	rating.ID = result.ID.String()
	rating.CreatedAt = result.CreatedAt.Time.UTC()
	rating.UpdatedAt = result.UpdatedAt.Time.UTC()

	return nil
}

// GetStoreRatingByID method
func (repo *StoreRatingRepository) GetStoreRatingByID(ctx context.Context, id string) (*domain.StoreRating, error) {
	var ratingID pgtype.UUID

	if err := ratingID.Scan(id); err != nil {
		return nil, fmt.Errorf("invalid ratingID: %w", domain.ErrInvalidInput)
	}

	result, err := repo.queries.GetStoreRatingByID(ctx, ratingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("store rating with id %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("getting store rating %s: %w", id, err)
	}

	return toDomainStoreRating(result), nil
}

// GetStoreRatingByAppointmentID method
func (repo *StoreRatingRepository) GetStoreRatingByAppointmentID(ctx context.Context, appointmentID string) (*domain.StoreRating, error) {
	var aID pgtype.UUID

	if err := aID.Scan(appointmentID); err != nil {
		return nil, fmt.Errorf("invalid appointmentID: %w", domain.ErrInvalidInput)
	}

	result, err := repo.queries.GetStoreRatingByAppointmentID(ctx, aID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("store rating for appointment %s: %w", appointmentID, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("getting store rating by appointment %s: %w", appointmentID, err)
	}

	return toDomainStoreRating(result), nil
}

// ListStoreRatingsByStore method
func (repo *StoreRatingRepository) ListStoreRatingsByStore(ctx context.Context, storeID string, limit, offset int32) ([]*domain.StoreRatingWithClient, error) {
	var sID pgtype.UUID

	if err := sID.Scan(storeID); err != nil {
		return nil, fmt.Errorf("invalid storeID: %w", domain.ErrInvalidInput)
	}

	results, err := repo.queries.ListStoreRatingsByStore(ctx, postgres.ListStoreRatingsByStoreParams{
		StoreID: sID,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, fmt.Errorf("listing store ratings: %w", err)
	}

	ratings := make([]*domain.StoreRatingWithClient, 0, len(results))
	for _, result := range results {
		var comment string
		if result.Comment != nil {
			comment = *result.Comment
		}

		ratings = append(ratings, &domain.StoreRatingWithClient{
			StoreRating: domain.StoreRating{
				ID:            result.ID.String(),
				AppointmentID: result.AppointmentID.String(),
				StoreID:       result.StoreID.String(),
				UserID:        result.ClientID.String(),
				Score:         int32(result.Score),
				Comment:       comment,
				CreatedAt:     result.CreatedAt.Time.UTC(),
				UpdatedAt:     result.UpdatedAt.Time.UTC(),
			},
			ClientUsername:  result.ClientUsername,
			ClientFirstName: result.ClientFirstName,
			ClientLastName:  result.ClientLastName,
		})
	}

	return ratings, nil
}

// CountStoreRatingsByStore method
func (repo *StoreRatingRepository) CountStoreRatingsByStore(ctx context.Context, storeID string) (int64, error) {
	var sID pgtype.UUID

	if err := sID.Scan(storeID); err != nil {
		return 0, fmt.Errorf("invalid storeID: %w", domain.ErrInvalidInput)
	}

	count, err := repo.queries.CountStoreRatingsByStore(ctx, sID)
	if err != nil {
		return 0, fmt.Errorf("counting store ratings: %w", err)
	}

	return count, nil
}

// GetStoreRatingSummaryByStore method
func (repo *StoreRatingRepository) GetStoreRatingSummaryByStore(ctx context.Context, storeID string) (*domain.StoreRatingSummary, error) {
	var sID pgtype.UUID

	if err := sID.Scan(storeID); err != nil {
		return nil, fmt.Errorf("invalid storeID: %w", domain.ErrInvalidInput)
	}

	result, err := repo.queries.GetStoreRatingSummaryByStore(ctx, sID)
	if err != nil {
		return nil, fmt.Errorf("getting store rating summary: %w", err)
	}

	return &domain.StoreRatingSummary{
		StoreID:      storeID,
		AverageScore: result.AverageScore,
		RatingCount:  result.RatingCount,
	}, nil
}

// ListRatedAppointmentIDs method returns the subset of appointmentIDs that already have a
// rating, in a single query. An appointmentID with no rating is simply absent from the result.
func (repo *StoreRatingRepository) ListRatedAppointmentIDs(ctx context.Context, appointmentIDs []string) ([]string, error) {
	ids := make([]pgtype.UUID, 0, len(appointmentIDs))
	for _, appointmentID := range appointmentIDs {
		var id pgtype.UUID
		if err := id.Scan(appointmentID); err != nil {
			return nil, fmt.Errorf("invalid appointmentID %q: %w", appointmentID, domain.ErrInvalidInput)
		}
		ids = append(ids, id)
	}

	results, err := repo.queries.ListRatedAppointmentIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("listing rated appointment IDs: %w", err)
	}

	rated := make([]string, 0, len(results))
	for _, result := range results {
		rated = append(rated, result.String())
	}

	return rated, nil
}

// GetStoreRatingSummariesByStores method returns the average score and rating count for each of
// storeIDs that has at least one rating, in a single query. A storeID with zero ratings is
// simply absent from the result; the caller treats a missing entry as a zero summary.
func (repo *StoreRatingRepository) GetStoreRatingSummariesByStores(ctx context.Context, storeIDs []string) ([]*domain.StoreRatingSummary, error) {
	ids := make([]pgtype.UUID, 0, len(storeIDs))
	for _, storeID := range storeIDs {
		var id pgtype.UUID
		if err := id.Scan(storeID); err != nil {
			return nil, fmt.Errorf("invalid storeID %q: %w", storeID, domain.ErrInvalidInput)
		}
		ids = append(ids, id)
	}

	results, err := repo.queries.GetStoreRatingSummariesByStores(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("getting store rating summaries: %w", err)
	}

	summaries := make([]*domain.StoreRatingSummary, 0, len(results))
	for _, result := range results {
		summaries = append(summaries, &domain.StoreRatingSummary{
			StoreID:      result.StoreID.String(),
			AverageScore: result.AverageScore,
			RatingCount:  result.RatingCount,
		})
	}

	return summaries, nil
}

// UpdateStoreRating method
func (repo *StoreRatingRepository) UpdateStoreRating(ctx context.Context, id string, score int32, comment string) (*domain.StoreRating, error) {
	var ratingID pgtype.UUID

	if err := ratingID.Scan(id); err != nil {
		return nil, fmt.Errorf("invalid ratingID: %w", domain.ErrInvalidInput)
	}

	var commentArg *string
	if comment != "" {
		commentArg = &comment
	}

	result, err := repo.queries.UpdateStoreRating(ctx, postgres.UpdateStoreRatingParams{
		ID:      ratingID,
		Score:   int16(score),
		Comment: commentArg,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("store rating with id %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("updating store rating %s: %w", id, err)
	}

	return toDomainStoreRating(result), nil
}

// DeleteStoreRating method
func (repo *StoreRatingRepository) DeleteStoreRating(ctx context.Context, id string) error {
	var ratingID pgtype.UUID

	if err := ratingID.Scan(id); err != nil {
		return fmt.Errorf("invalid ratingID: %w", domain.ErrInvalidInput)
	}

	if err := repo.queries.DeleteStoreRating(ctx, ratingID); err != nil {
		return fmt.Errorf("deleting store rating %s: %w", id, err)
	}

	return nil
}
