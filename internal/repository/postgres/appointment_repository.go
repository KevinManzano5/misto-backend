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

// AppointmentRepository struct
type AppointmentRepository struct {
	pool    *pgxpool.Pool
	queries *postgres.Queries
}

// toDomainAppointment function
//
// Timestamps are normalized to UTC here, the single point where appointments cross from
// Postgres into the domain, so every layer above works with a consistent timezone.
func toDomainAppointment(appointment postgres.Appointment) *domain.Appointment {
	return &domain.Appointment{
		ID:        appointment.ID.String(),
		StoreID:   appointment.StoreID.String(),
		UserID:    appointment.ClientID.String(),
		StartTime: appointment.StartTime.Time.UTC(),
		EndTime:   appointment.EndTime.Time.UTC(),
		Status:    appointment.Status,
		CreatedAt: appointment.CreatedAt.Time.UTC(),
		UpdatedAt: appointment.UpdatedAt.Time.UTC(),
	}
}

// NewAppointmentRepository function
func NewAppointmentRepository(pool *pgxpool.Pool) *AppointmentRepository {
	return &AppointmentRepository{
		pool:    pool,
		queries: postgres.New(pool),
	}
}

// CreateAppointment method atomically checks the requested time slot for a conflict and
// inserts the appointment within a single transaction, so a concurrent request for the same
// slot can never slip past the overlap check before either insert commits.
func (repo *AppointmentRepository) CreateAppointment(ctx context.Context, appointment *domain.Appointment) error {
	var storeID, clientID pgtype.UUID

	if err := storeID.Scan(appointment.StoreID); err != nil {
		return fmt.Errorf("invalid storeID: %w", domain.ErrInvalidInput)
	}
	if err := clientID.Scan(appointment.UserID); err != nil {
		return fmt.Errorf("invalid userID: %w", domain.ErrInvalidInput)
	}

	start := pgtype.Timestamptz{Time: appointment.StartTime, Valid: true}
	end := pgtype.Timestamptz{Time: appointment.EndTime, Valid: true}

	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning appointment transaction: %w", err)
	}
	defer func() {
		// Rollback is a safe no-op (pgx.ErrTxClosed) once the transaction has already been committed below.
		_ = tx.Rollback(ctx)
	}()

	txQueries := repo.queries.WithTx(tx)

	count, err := txQueries.CheckAppointmentOverlap(ctx, postgres.CheckAppointmentOverlapParams{
		StoreID:      storeID,
		NewStartTime: start,
		NewEndTime:   end,
	})
	if err != nil {
		return fmt.Errorf("checking appointment overlap: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("%w: time slot already booked", domain.ErrAlreadyExists)
	}

	result, err := txQueries.CreateAppointment(ctx, postgres.CreateAppointmentParams{
		StoreID:   storeID,
		ClientID:  clientID,
		StartTime: start,
		EndTime:   end,
	})
	if err != nil {
		return translateAppointmentWriteError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing appointment creation: %w", err)
	}

	appointment.ID = result.ID.String()
	appointment.Status = result.Status
	appointment.StartTime = result.StartTime.Time.UTC()
	appointment.EndTime = result.EndTime.Time.UTC()
	appointment.CreatedAt = result.CreatedAt.Time.UTC()
	appointment.UpdatedAt = result.UpdatedAt.Time.UTC()

	return nil
}

// translateAppointmentWriteError maps known Postgres constraint violations from inserting an
// appointment to the shared domain sentinel errors.
func translateAppointmentWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503": // foreign_key_violation: store_id or client_id does not reference an existing row
			return fmt.Errorf("%w: referenced store or client does not exist", domain.ErrInvalidInput)
		case "23505": // unique_violation
			return fmt.Errorf("%w", domain.ErrAlreadyExists)
		}
	}

	return fmt.Errorf("creating appointment: %w", err)
}

// ListAppointmentsByStoreDate method
func (repo *AppointmentRepository) ListAppointmentsByStoreDate(ctx context.Context, storeID string, startTime, endTime time.Time) ([]*domain.Appointment, error) {
	var sID pgtype.UUID

	if err := sID.Scan(storeID); err != nil {
		return nil, fmt.Errorf("invalid storeID: %w", domain.ErrInvalidInput)
	}

	start := pgtype.Timestamptz{Time: startTime, Valid: true}
	end := pgtype.Timestamptz{Time: endTime, Valid: true}

	results, err := repo.queries.ListAppointmentsByStoreDate(ctx, postgres.ListAppointmentsByStoreDateParams{
		StoreID:     sID,
		StartTime:   start,
		StartTime_2: end,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []*domain.Appointment{}, nil // If there are no appointments, we return an empty slice, not an error
		}
		return nil, fmt.Errorf("listing appointments: %w", err)
	}

	var appointments []*domain.Appointment
	for _, result := range results {
		appointments = append(appointments, toDomainAppointment(result))
	}

	return appointments, nil
}
