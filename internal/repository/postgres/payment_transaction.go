package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/KevinManzano5/misto-backend/internal/domain"
	postgres "github.com/KevinManzano5/misto-backend/internal/repository/postgres/generated"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PaymentTransactionRepository struct
type PaymentTransactionRepository struct {
	pool    *pgxpool.Pool
	queries *postgres.Queries
}

// NewPaymentTransactionRepository function
func NewPaymentTransactionRepository(pool *pgxpool.Pool) *PaymentTransactionRepository {
	return &PaymentTransactionRepository{
		pool:    pool,
		queries: postgres.New(pool),
	}
}

// toDomainPaymentTransaction function
func toDomainPaymentTransaction(record postgres.PaymentTransaction) *domain.PaymentTransaction {
	var failureReason string
	if record.FailureReason != nil {
		failureReason = *record.FailureReason
	}

	return &domain.PaymentTransaction{
		ID:            record.ID.String(),
		AppointmentID: record.AppointmentID.String(),
		Amount:        record.Amount,
		Currency:      record.Currency,
		Status:        record.Status,
		PaymentMethod: record.PaymentMethod,
		FailureReason: failureReason,
		CreatedAt:     record.CreatedAt.Time.UTC(),
	}
}

// translatePaymentTransactionWriteError maps known Postgres constraint violations from writing a
// payment transaction to the shared domain sentinel errors.
func translatePaymentTransactionWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23503" { // foreign_key_violation: appointment_id does not reference an existing row
			return fmt.Errorf("%w: referenced appointment does not exist", domain.ErrInvalidInput)
		}
	}

	return fmt.Errorf("writing payment transaction: %w", err)
}

// CreatePaymentTransaction method
func (repo *PaymentTransactionRepository) CreatePaymentTransaction(ctx context.Context, record *domain.PaymentTransaction) error {
	var appointmentID pgtype.UUID

	if err := appointmentID.Scan(record.AppointmentID); err != nil {
		return fmt.Errorf("invalid appointmentID: %w", domain.ErrInvalidInput)
	}

	var failureReason *string
	if record.FailureReason != "" {
		failureReason = &record.FailureReason
	}

	result, err := repo.queries.CreatePaymentTransaction(ctx, postgres.CreatePaymentTransactionParams{
		AppointmentID: appointmentID,
		Amount:        record.Amount,
		Currency:      record.Currency,
		Status:        record.Status,
		PaymentMethod: record.PaymentMethod,
		FailureReason: failureReason,
	})
	if err != nil {
		return translatePaymentTransactionWriteError(err)
	}

	record.ID = result.ID.String()
	record.CreatedAt = result.CreatedAt.Time.UTC()

	return nil
}

// ListPaymentTransactionsByAppointment method
func (repo *PaymentTransactionRepository) ListPaymentTransactionsByAppointment(ctx context.Context, appointmentID string) ([]*domain.PaymentTransaction, error) {
	var aID pgtype.UUID

	if err := aID.Scan(appointmentID); err != nil {
		return nil, fmt.Errorf("invalid appointmentID: %w", domain.ErrInvalidInput)
	}

	results, err := repo.queries.ListPaymentTransactionsByAppointment(ctx, aID)
	if err != nil {
		return nil, fmt.Errorf("listing payment transactions: %w", err)
	}

	transactions := make([]*domain.PaymentTransaction, 0, len(results))
	for _, result := range results {
		transactions = append(transactions, toDomainPaymentTransaction(result))
	}

	return transactions, nil
}
