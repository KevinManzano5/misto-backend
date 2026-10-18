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

// AppointmentRepository struct
type AppointmentRepository struct {
	pool    *pgxpool.Pool
	queries *postgres.Queries
}

// toDomainAppointment function
func toDomainAppointment(appointment postgres.Appointment) *domain.Appointment {
	return &domain.Appointment{
		ID:        appointment.ID.String(),
		StoreID:   appointment.StoreID.String(),
		ServiceID: appointment.ServiceID.String(),
		UserID:    appointment.ClientID.String(),
		StartTime: appointment.StartTime.Time,
		EndTime:   appointment.EndTime.Time,
		Status:    appointment.Status,
		CreatedAt: appointment.CreatedAt.Time,
		UpdatedAt: appointment.UpdatedAt.Time,
	}
}

// NewAppointmentRepository function
func NewAppointmentRepository(pool *pgxpool.Pool) *AppointmentRepository {
	return &AppointmentRepository{
		pool:    pool,
		queries: postgres.New(pool),
	}
}

// CheckOverlap method
// Returns true if there is an appointment that overlaps with the specified time slot.
func (repo *AppointmentRepository) CheckOverlap(ctx context.Context, storeID string, startTime, endTime time.Time) (bool, error) {
	var sID pgtype.UUID

	if err := sID.Scan(storeID); err != nil {
		return false, fmt.Errorf("invalid storeID: %w", domain.ErrInvalidInput)
	}

	start := pgtype.Timestamptz{Time: startTime, Valid: true}
	end := pgtype.Timestamptz{Time: endTime, Valid: true}

	count, err := repo.queries.CheckAppointmentOverlap(ctx, postgres.CheckAppointmentOverlapParams{
		StoreID:   sID,
		StartTime: start,
		EndTime:   end,
	})

	if err != nil {
		return false, fmt.Errorf("checking appointment overlap: %w", err)
	}

	return count > 0, nil
}

// CreateAppointment method
func (repo *AppointmentRepository) CreateAppointment(ctx context.Context, appointment *domain.Appointment) error {
	var storeID, serviceID, clientID pgtype.UUID

	if err := storeID.Scan(appointment.StoreID); err != nil {
		return fmt.Errorf("invalid storeID: %w", domain.ErrInvalidInput)
	}
	if err := serviceID.Scan(appointment.ServiceID); err != nil {
		return fmt.Errorf("invalid serviceID: %w", domain.ErrInvalidInput)
	}
	if err := clientID.Scan(appointment.UserID); err != nil {
		return fmt.Errorf("invalid userID: %w", domain.ErrInvalidInput)
	}

	start := pgtype.Timestamptz{Time: appointment.StartTime, Valid: true}
	end := pgtype.Timestamptz{Time: appointment.EndTime, Valid: true}

	result, err := repo.queries.CreateAppointment(ctx, postgres.CreateAppointmentParams{
		StoreID:   storeID,
		ServiceID: serviceID,
		ClientID:  clientID,
		StartTime: start,
		EndTime:   end,
	})

	if err != nil {
		return fmt.Errorf("creating appointment: %w", err)
	}

	appointment.ID = result.ID.String()
	appointment.Status = result.Status
	appointment.CreatedAt = result.CreatedAt.Time
	appointment.UpdatedAt = result.UpdatedAt.Time

	return nil
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
