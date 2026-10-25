package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/KevinManzano5/misto-backend/internal/domain"
	postgres "github.com/KevinManzano5/misto-backend/internal/repository/postgres/generated"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StoreScheduleRepository struct
type StoreScheduleRepository struct {
	pool    *pgxpool.Pool
	queries *postgres.Queries
}

// NewStoreScheduleRepository function
func NewStoreScheduleRepository(pool *pgxpool.Pool) *StoreScheduleRepository {
	return &StoreScheduleRepository{
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

// toDomainStoreSchedule function
func toDomainStoreSchedule(schedule postgres.StoreSchedule) *domain.StoreSchedule {
	return &domain.StoreSchedule{
		ID:        schedule.ID.String(),
		StoreID:   schedule.StoreID.String(),
		DayOfWeek: schedule.DayOfWeek,
		OpenTime:  fromPgTime(schedule.OpenTime),
		CloseTime: fromPgTime(schedule.CloseTime),
		IsClosed:  schedule.IsClosed,
		CreatedAt: schedule.CreatedAt.Time,
		UpdatedAt: schedule.UpdatedAt.Time,
	}
}

// UpsertStoreSchedule method inserts or updates the schedule for a specific day
func (repo *StoreScheduleRepository) UpsertStoreSchedule(ctx context.Context, schedule *domain.StoreSchedule) error {
	var storeID pgtype.UUID

	if err := storeID.Scan(schedule.StoreID); err != nil {
		return fmt.Errorf("invalid storeID: %w", domain.ErrInvalidInput)
	}

	result, err := repo.queries.UpsertStoreSchedule(ctx, postgres.UpsertStoreScheduleParams{
		StoreID:   storeID,
		DayOfWeek: schedule.DayOfWeek,
		OpenTime:  toPgTime(schedule.OpenTime),
		CloseTime: toPgTime(schedule.CloseTime),
		IsClosed:  schedule.IsClosed,
	})

	if err != nil {
		return fmt.Errorf("upserting store schedule: %w", err)
	}

	schedule.ID = result.ID.String()
	schedule.CreatedAt = result.CreatedAt.Time
	schedule.UpdatedAt = result.UpdatedAt.Time
	schedule.OpenTime = fromPgTime(result.OpenTime)
	schedule.CloseTime = fromPgTime(result.CloseTime)

	return nil
}

// GetStoreScheduleByDay method retrieves the schedule of a store for a specific day
func (repo *StoreScheduleRepository) GetStoreScheduleByDay(ctx context.Context, storeID string, dayOfWeek int32) (*domain.StoreSchedule, error) {
	var sID pgtype.UUID

	if err := sID.Scan(storeID); err != nil {
		return nil, fmt.Errorf("invalid storeID: %w", domain.ErrInvalidInput)
	}

	result, err := repo.queries.GetStoreScheduleByDay(ctx, postgres.GetStoreScheduleByDayParams{
		StoreID:   sID,
		DayOfWeek: dayOfWeek,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("schedule for day %d: %w", dayOfWeek, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("getting store schedule: %w", err)
	}

	return toDomainStoreSchedule(result), nil
}

// ListStoreSchedules method retrieves all schedules for a specific store
func (repo *StoreScheduleRepository) ListStoreSchedules(ctx context.Context, storeID string) ([]*domain.StoreSchedule, error) {
	var sID pgtype.UUID

	if err := sID.Scan(storeID); err != nil {
		return nil, fmt.Errorf("invalid storeID: %w", domain.ErrInvalidInput)
	}

	results, err := repo.queries.ListStoreSchedules(ctx, sID)
	if err != nil {
		return nil, fmt.Errorf("listing store schedules: %w", err)
	}

	schedules := make([]*domain.StoreSchedule, 0)
	for _, result := range results {
		schedules = append(schedules, toDomainStoreSchedule(result))
	}

	return schedules, nil
}
