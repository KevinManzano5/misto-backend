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

// CreateAppointment method atomically checks that the client has no other pending/confirmed
// appointment at this store and that the requested time slot has no conflict, then inserts the
// appointment, all within a single transaction, so a concurrent request can never slip past
// either check before either insert commits.
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

	activeCount, err := txQueries.CheckClientHasActiveAppointment(ctx, postgres.CheckClientHasActiveAppointmentParams{
		StoreID:  storeID,
		ClientID: clientID,
	})
	if err != nil {
		return fmt.Errorf("checking client active appointment: %w", err)
	}
	if activeCount > 0 {
		return fmt.Errorf("%w: client already has a pending or confirmed appointment at this store", domain.ErrAlreadyExists)
	}

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
		Status:    appointment.Status,
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

// ListAllAppointmentsByStoreWithClient method returns every appointment ever booked at storeID,
// any status, most recent first, each paired with the basic profile of the client who booked it.
// startTime/endTime optionally restrict the results to an inclusive start_time range; either or
// both may be nil to leave that side of the range open.
func (repo *AppointmentRepository) ListAllAppointmentsByStoreWithClient(ctx context.Context, storeID string, startTime, endTime *time.Time) ([]*domain.AppointmentWithClient, error) {
	var sID pgtype.UUID

	if err := sID.Scan(storeID); err != nil {
		return nil, fmt.Errorf("invalid storeID: %w", domain.ErrInvalidInput)
	}

	var start, end pgtype.Timestamptz
	if startTime != nil {
		start = pgtype.Timestamptz{Time: *startTime, Valid: true}
	}
	if endTime != nil {
		end = pgtype.Timestamptz{Time: *endTime, Valid: true}
	}

	results, err := repo.queries.ListAllAppointmentsByStoreWithClient(ctx, postgres.ListAllAppointmentsByStoreWithClientParams{
		StoreID:   sID,
		StartDate: start,
		EndDate:   end,
	})
	if err != nil {
		return nil, fmt.Errorf("listing all appointments: %w", err)
	}

	appointments := make([]*domain.AppointmentWithClient, 0, len(results))
	for _, result := range results {
		appointments = append(appointments, &domain.AppointmentWithClient{
			Appointment: domain.Appointment{
				ID:        result.ID.String(),
				StoreID:   result.StoreID.String(),
				UserID:    result.ClientID.String(),
				StartTime: result.StartTime.Time.UTC(),
				EndTime:   result.EndTime.Time.UTC(),
				Status:    result.Status,
				CreatedAt: result.CreatedAt.Time.UTC(),
				UpdatedAt: result.UpdatedAt.Time.UTC(),
			},
			ClientUsername:  result.ClientUsername,
			ClientEmail:     result.ClientEmail,
			ClientFirstName: result.ClientFirstName,
			ClientLastName:  result.ClientLastName,
		})
	}

	return appointments, nil
}

// ListAppointmentsByClient method returns every appointment ever booked by clientID, any status,
// most recent first, each paired with the basic info of the store it was booked at.
func (repo *AppointmentRepository) ListAppointmentsByClient(ctx context.Context, clientID string) ([]*domain.AppointmentWithStore, error) {
	var cID pgtype.UUID

	if err := cID.Scan(clientID); err != nil {
		return nil, fmt.Errorf("invalid clientID: %w", domain.ErrInvalidInput)
	}

	results, err := repo.queries.ListAppointmentsByClient(ctx, cID)
	if err != nil {
		return nil, fmt.Errorf("listing appointments by client: %w", err)
	}

	appointments := make([]*domain.AppointmentWithStore, 0, len(results))
	for _, result := range results {
		appointments = append(appointments, &domain.AppointmentWithStore{
			Appointment: domain.Appointment{
				ID:        result.ID.String(),
				StoreID:   result.StoreID.String(),
				UserID:    result.ClientID.String(),
				StartTime: result.StartTime.Time.UTC(),
				EndTime:   result.EndTime.Time.UTC(),
				Status:    result.Status,
				CreatedAt: result.CreatedAt.Time.UTC(),
				UpdatedAt: result.UpdatedAt.Time.UTC(),
			},
			StoreName: result.StoreName,
		})
	}

	return appointments, nil
}

// GetAppointmentByID method
func (repo *AppointmentRepository) GetAppointmentByID(ctx context.Context, id string) (*domain.Appointment, error) {
	var appointmentID pgtype.UUID

	if err := appointmentID.Scan(id); err != nil {
		return nil, fmt.Errorf("invalid appointmentID: %w", domain.ErrInvalidInput)
	}

	result, err := repo.queries.GetAppointmentByID(ctx, appointmentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("appointment with id %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("getting appointment %s: %w", id, err)
	}

	return toDomainAppointment(result), nil
}

// ExpireOverdueAppointments moves pending/confirmed appointments whose end_time is before
// `before` to a terminal status (completed/expired) in a single bulk update, and returns the
// rows that were changed.
func (repo *AppointmentRepository) ExpireOverdueAppointments(ctx context.Context, before time.Time) ([]*domain.Appointment, error) {
	results, err := repo.queries.ExpireOverdueAppointments(ctx, pgtype.Timestamptz{Time: before, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("expiring overdue appointments: %w", err)
	}

	appointments := make([]*domain.Appointment, 0, len(results))
	for _, result := range results {
		appointments = append(appointments, toDomainAppointment(result))
	}

	return appointments, nil
}

// UpdateAppointmentStatus method
func (repo *AppointmentRepository) UpdateAppointmentStatus(ctx context.Context, id, status string) (*domain.Appointment, error) {
	var appointmentID pgtype.UUID

	if err := appointmentID.Scan(id); err != nil {
		return nil, fmt.Errorf("invalid appointmentID: %w", domain.ErrInvalidInput)
	}

	result, err := repo.queries.UpdateAppointmentStatus(ctx, postgres.UpdateAppointmentStatusParams{
		ID:     appointmentID,
		Status: status,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("appointment with id %s: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("updating appointment %s: %w", id, err)
	}

	return toDomainAppointment(result), nil
}
