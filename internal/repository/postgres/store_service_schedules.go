package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/KevinManzano5/misto-backend/internal/domain"
	postgres "github.com/KevinManzano5/misto-backend/internal/repository/postgres/generated"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StoreServiceScheduleRepository struct
type StoreServiceScheduleRepository struct {
	pool    *pgxpool.Pool
	queries *postgres.Queries
}

// NewStoreServiceScheduleRepository function
func NewStoreServiceScheduleRepository(pool *pgxpool.Pool) *StoreServiceScheduleRepository {
	return &StoreServiceScheduleRepository{
		pool:    pool,
		queries: postgres.New(pool),
	}
}

// toPgTime helper function to convert string HH:MM or HH:MM:SS to pgtype.Time
func toPgTime(timeStr string) pgtype.Time {
	if timeStr == "" {
		return pgtype.Time{Valid: false}
	}

	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		t, err = time.Parse("15:04:05", timeStr)
		if err != nil {
			return pgtype.Time{Valid: false}
		}
	}

	microseconds := int64(t.Hour()*3600+t.Minute()*60+t.Second()) * 1000000
	return pgtype.Time{Microseconds: microseconds, Valid: true}
}

// fromPgTime helper function to convert pgtype.Time to string HH:MM:SS
func fromPgTime(t pgtype.Time) string {
	if !t.Valid {
		return ""
	}
	seconds := t.Microseconds / 1000000
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}

// toDomainStoreServiceScheduleDay converts a store_service_schedules row (without segments) to a
// partially filled domain.StoreServiceSchedule; callers attach Segments afterwards.
func toDomainStoreServiceScheduleDay(day postgres.StoreServiceSchedule) *domain.StoreServiceSchedule {
	return &domain.StoreServiceSchedule{
		ID:             day.ID.String(),
		StoreServiceID: day.StoreServiceID.String(),
		DayOfWeek:      day.DayOfWeek,
		IsClosed:       day.IsClosed,
		CreatedAt:      day.CreatedAt.Time.UTC(),
		UpdatedAt:      day.UpdatedAt.Time.UTC(),
	}
}

// toDomainServiceSegment converts a store_service_schedule_segments row to a
// domain.StoreServiceScheduleSegment
func toDomainServiceSegment(segment postgres.StoreServiceScheduleSegment) domain.StoreServiceScheduleSegment {
	return domain.StoreServiceScheduleSegment{
		ID:        segment.ID.String(),
		OpenTime:  fromPgTime(segment.OpenTime),
		CloseTime: fromPgTime(segment.CloseTime),
	}
}

// translateServiceScheduleSegmentWriteError maps the open_time < close_time check constraint to
// domain.ErrInvalidInput, so the usecase never needs to inspect pgconn.PgError.
func translateServiceScheduleSegmentWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23514" { // check_violation
		return fmt.Errorf("%w: segment open_time must be before close_time", domain.ErrInvalidInput)
	}

	return fmt.Errorf("writing store service schedule segment: %w", err)
}

// UpsertStoreServiceSchedule method atomically replaces the day's config and its full set of
// segments: it upserts the day row, deletes its previous segments and inserts the new ones, so a
// failure inserting any segment rolls back instead of leaving the day without segments.
func (repo *StoreServiceScheduleRepository) UpsertStoreServiceSchedule(ctx context.Context, schedule *domain.StoreServiceSchedule) error {
	var storeServiceID pgtype.UUID

	if err := storeServiceID.Scan(schedule.StoreServiceID); err != nil {
		return fmt.Errorf("invalid storeServiceID: %w", domain.ErrInvalidInput)
	}

	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning store service schedule transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txQueries := repo.queries.WithTx(tx)

	dayResult, err := txQueries.UpsertStoreServiceScheduleDay(ctx, postgres.UpsertStoreServiceScheduleDayParams{
		StoreServiceID: storeServiceID,
		DayOfWeek:      schedule.DayOfWeek,
		IsClosed:       schedule.IsClosed,
	})
	if err != nil {
		return fmt.Errorf("upserting store service schedule day: %w", err)
	}

	if err := txQueries.DeleteStoreServiceScheduleSegments(ctx, dayResult.ID); err != nil {
		return fmt.Errorf("clearing store service schedule segments: %w", err)
	}

	segments := make([]domain.StoreServiceScheduleSegment, 0, len(schedule.Segments))
	for _, segment := range schedule.Segments {
		segmentResult, err := txQueries.CreateStoreServiceScheduleSegment(ctx, postgres.CreateStoreServiceScheduleSegmentParams{
			StoreServiceScheduleID: dayResult.ID,
			OpenTime:               toPgTime(segment.OpenTime),
			CloseTime:              toPgTime(segment.CloseTime),
		})
		if err != nil {
			return translateServiceScheduleSegmentWriteError(err)
		}
		segments = append(segments, toDomainServiceSegment(segmentResult))
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing store service schedule: %w", err)
	}

	schedule.ID = dayResult.ID.String()
	schedule.CreatedAt = dayResult.CreatedAt.Time.UTC()
	schedule.UpdatedAt = dayResult.UpdatedAt.Time.UTC()
	schedule.Segments = segments

	return nil
}

// GetStoreServiceScheduleByDay method retrieves the schedule (with its segments) of a store
// service for a specific day
func (repo *StoreServiceScheduleRepository) GetStoreServiceScheduleByDay(ctx context.Context, storeServiceID string, dayOfWeek int32) (*domain.StoreServiceSchedule, error) {
	var sID pgtype.UUID

	if err := sID.Scan(storeServiceID); err != nil {
		return nil, fmt.Errorf("invalid storeServiceID: %w", domain.ErrInvalidInput)
	}

	dayResult, err := repo.queries.GetStoreServiceScheduleDay(ctx, postgres.GetStoreServiceScheduleDayParams{
		StoreServiceID: sID,
		DayOfWeek:      dayOfWeek,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("schedule for day %d: %w", dayOfWeek, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("getting store service schedule: %w", err)
	}

	segmentResults, err := repo.queries.ListStoreServiceScheduleSegments(ctx, dayResult.ID)
	if err != nil {
		return nil, fmt.Errorf("listing store service schedule segments: %w", err)
	}

	schedule := toDomainStoreServiceScheduleDay(dayResult)
	schedule.Segments = make([]domain.StoreServiceScheduleSegment, 0, len(segmentResults))
	for _, segment := range segmentResults {
		schedule.Segments = append(schedule.Segments, toDomainServiceSegment(segment))
	}

	return schedule, nil
}

// ListStoreServiceSchedules method retrieves all schedules (each with its segments) for a
// specific store service
func (repo *StoreServiceScheduleRepository) ListStoreServiceSchedules(ctx context.Context, storeServiceID string) ([]*domain.StoreServiceSchedule, error) {
	var sID pgtype.UUID

	if err := sID.Scan(storeServiceID); err != nil {
		return nil, fmt.Errorf("invalid storeServiceID: %w", domain.ErrInvalidInput)
	}

	dayResults, err := repo.queries.ListStoreServiceScheduleDays(ctx, sID)
	if err != nil {
		return nil, fmt.Errorf("listing store service schedules: %w", err)
	}

	scheduleIDs := make([]pgtype.UUID, 0, len(dayResults))
	schedules := make([]*domain.StoreServiceSchedule, 0, len(dayResults))
	byScheduleID := make(map[string]*domain.StoreServiceSchedule, len(dayResults))

	for _, day := range dayResults {
		schedule := toDomainStoreServiceScheduleDay(day)
		schedule.Segments = make([]domain.StoreServiceScheduleSegment, 0)
		schedules = append(schedules, schedule)
		byScheduleID[schedule.ID] = schedule
		scheduleIDs = append(scheduleIDs, day.ID)
	}

	segmentResults, err := repo.queries.ListStoreServiceScheduleSegmentsByScheduleIDs(ctx, scheduleIDs)
	if err != nil {
		return nil, fmt.Errorf("listing store service schedule segments: %w", err)
	}

	for _, segment := range segmentResults {
		if schedule, ok := byScheduleID[segment.StoreServiceScheduleID.String()]; ok {
			schedule.Segments = append(schedule.Segments, toDomainServiceSegment(segment))
		}
	}

	return schedules, nil
}
