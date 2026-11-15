package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/KevinManzano5/misto-backend/internal/domain"
	postgres "github.com/KevinManzano5/misto-backend/internal/repository/postgres/generated"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DebtRepository struct
type DebtRepository struct {
	pool    *pgxpool.Pool
	queries *postgres.Queries
}

// NewDebtRepository function
func NewDebtRepository(pool *pgxpool.Pool) *DebtRepository {
	return &DebtRepository{
		pool:    pool,
		queries: postgres.New(pool),
	}
}

// toDomainUserDebt function
func toDomainUserDebt(userID, storeID string, debt postgres.UserDebt) *domain.UserDebt {
	return &domain.UserDebt{
		ID:        debt.ID.String(),
		UserID:    userID,
		StoreID:   storeID,
		Balance:   debt.Balance,
		CreatedAt: debt.CreatedAt.Time.UTC(),
		UpdatedAt: debt.UpdatedAt.Time.UTC(),
	}
}

// toDomainDebtTransaction function
func toDomainDebtTransaction(transaction postgres.DebtTransaction) *domain.DebtTransaction {
	var appointmentID string
	if transaction.AppointmentID.Valid {
		appointmentID = transaction.AppointmentID.String()
	}

	return &domain.DebtTransaction{
		ID:            transaction.ID.String(),
		UserID:        transaction.UserID.String(),
		StoreID:       transaction.StoreID.String(),
		AppointmentID: appointmentID,
		Amount:        transaction.Amount,
		Type:          transaction.Type,
		CreatedAt:     transaction.CreatedAt.Time.UTC(),
	}
}

// scanUUIDs parses userID/storeID into pgtype.UUID, returning domain.ErrInvalidInput on failure.
func scanUUIDs(userID, storeID string) (userUUID, storeUUID pgtype.UUID, err error) {
	if err := userUUID.Scan(userID); err != nil {
		return userUUID, storeUUID, fmt.Errorf("invalid userID: %w", domain.ErrInvalidInput)
	}
	if err := storeUUID.Scan(storeID); err != nil {
		return userUUID, storeUUID, fmt.Errorf("invalid storeID: %w", domain.ErrInvalidInput)
	}
	return userUUID, storeUUID, nil
}

// GetUserDebt returns userID's balance owed to storeID. A user who has never owed this store
// anything is a normal, expected outcome: it returns a zero-balance UserDebt, not
// domain.ErrNotFound.
func (repo *DebtRepository) GetUserDebt(ctx context.Context, userID, storeID string) (*domain.UserDebt, error) {
	userUUID, storeUUID, err := scanUUIDs(userID, storeID)
	if err != nil {
		return nil, err
	}

	result, err := repo.queries.GetUserDebt(ctx, postgres.GetUserDebtParams{UserID: userUUID, StoreID: storeUUID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &domain.UserDebt{UserID: userID, StoreID: storeID, Balance: 0}, nil
		}
		return nil, fmt.Errorf("getting user debt: %w", err)
	}

	return toDomainUserDebt(userID, storeID, result), nil
}

// AddDebt opens or increases userID's debt with storeID by amount, atomically recording the
// 'charge' debt_transactions row that justifies it, because appointmentID's cash_on_site
// payment was never collected at booking time.
func (repo *DebtRepository) AddDebt(ctx context.Context, userID, storeID, appointmentID string, amount int64) error {
	userUUID, storeUUID, err := scanUUIDs(userID, storeID)
	if err != nil {
		return err
	}
	var appointmentUUID pgtype.UUID
	if err := appointmentUUID.Scan(appointmentID); err != nil {
		return fmt.Errorf("invalid appointmentID: %w", domain.ErrInvalidInput)
	}

	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning add debt transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txQueries := repo.queries.WithTx(tx)

	if _, err := txQueries.IncrementUserDebt(ctx, postgres.IncrementUserDebtParams{UserID: userUUID, StoreID: storeUUID, Balance: amount}); err != nil {
		return fmt.Errorf("incrementing user debt: %w", err)
	}

	if _, err := txQueries.CreateDebtTransaction(ctx, postgres.CreateDebtTransactionParams{
		UserID:        userUUID,
		StoreID:       storeUUID,
		AppointmentID: appointmentUUID,
		Amount:        amount,
		Type:          domain.DebtTransactionTypeCharge,
	}); err != nil {
		return fmt.Errorf("recording debt charge: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing add debt transaction: %w", err)
	}

	return nil
}

// ReverseDebt removes amount from userID's debt with storeID and records the matching
// 'reversal' debt_transactions row, because appointmentID (which had opened that debt) was
// cancelled before the money was ever collected.
func (repo *DebtRepository) ReverseDebt(ctx context.Context, userID, storeID, appointmentID string, amount int64) error {
	userUUID, storeUUID, err := scanUUIDs(userID, storeID)
	if err != nil {
		return err
	}
	var appointmentUUID pgtype.UUID
	if err := appointmentUUID.Scan(appointmentID); err != nil {
		return fmt.Errorf("invalid appointmentID: %w", domain.ErrInvalidInput)
	}

	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning reverse debt transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txQueries := repo.queries.WithTx(tx)

	if _, err := txQueries.DecrementUserDebt(ctx, postgres.DecrementUserDebtParams{UserID: userUUID, StoreID: storeUUID, Balance: amount}); err != nil {
		return fmt.Errorf("decrementing user debt: %w", err)
	}

	if _, err := txQueries.CreateDebtTransaction(ctx, postgres.CreateDebtTransactionParams{
		UserID:        userUUID,
		StoreID:       storeUUID,
		AppointmentID: appointmentUUID,
		Amount:        amount,
		Type:          domain.DebtTransactionTypeReversal,
	}); err != nil {
		return fmt.Errorf("recording debt reversal: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing reverse debt transaction: %w", err)
	}

	return nil
}

// SettleDebt confirms that appointmentID's cash_on_site debt (exactly amount, its own fixed
// price) was collected, atomically: it locks the balance row, decrements it by amount, records
// the matching 'payment' debt_transactions row tied to appointmentID, and marks the appointment
// itself as paid.
func (repo *DebtRepository) SettleDebt(ctx context.Context, userID, storeID, appointmentID string, amount int64) (*domain.Appointment, error) {
	userUUID, storeUUID, err := scanUUIDs(userID, storeID)
	if err != nil {
		return nil, err
	}
	var appointmentUUID pgtype.UUID
	if err := appointmentUUID.Scan(appointmentID); err != nil {
		return nil, fmt.Errorf("invalid appointmentID: %w", domain.ErrInvalidInput)
	}

	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning settle debt transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txQueries := repo.queries.WithTx(tx)

	current, err := txQueries.GetUserDebtForUpdate(ctx, postgres.GetUserDebtForUpdateParams{UserID: userUUID, StoreID: storeUUID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: user has no debt with this store", domain.ErrInvalidInput)
		}
		return nil, fmt.Errorf("locking user debt: %w", err)
	}
	if amount > current.Balance {
		return nil, fmt.Errorf("%w: appointment amount exceeds current debt balance", domain.ErrInvalidInput)
	}

	if _, err := txQueries.DecrementUserDebt(ctx, postgres.DecrementUserDebtParams{UserID: userUUID, StoreID: storeUUID, Balance: amount}); err != nil {
		return nil, fmt.Errorf("decrementing user debt: %w", err)
	}

	if _, err := txQueries.CreateDebtTransaction(ctx, postgres.CreateDebtTransactionParams{
		UserID:        userUUID,
		StoreID:       storeUUID,
		AppointmentID: appointmentUUID,
		Amount:        amount,
		Type:          domain.DebtTransactionTypePayment,
	}); err != nil {
		return nil, fmt.Errorf("recording debt payment: %w", err)
	}

	updatedAppointment, err := txQueries.UpdateAppointmentPaymentStatus(ctx, postgres.UpdateAppointmentPaymentStatusParams{ID: appointmentUUID, PaymentStatus: domain.PaymentStatusPaid})
	if err != nil {
		return nil, fmt.Errorf("marking appointment as paid: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing settle debt transaction: %w", err)
	}

	return toDomainAppointment(updatedAppointment), nil
}

// ListDebtTransactions returns the full debt history for userID at storeID, most recent first.
func (repo *DebtRepository) ListDebtTransactions(ctx context.Context, userID, storeID string) ([]*domain.DebtTransaction, error) {
	userUUID, storeUUID, err := scanUUIDs(userID, storeID)
	if err != nil {
		return nil, err
	}

	results, err := repo.queries.ListDebtTransactionsByUserStore(ctx, postgres.ListDebtTransactionsByUserStoreParams{UserID: userUUID, StoreID: storeUUID})
	if err != nil {
		return nil, fmt.Errorf("listing debt transactions: %w", err)
	}

	transactions := make([]*domain.DebtTransaction, 0, len(results))
	for _, result := range results {
		transactions = append(transactions, toDomainDebtTransaction(result))
	}

	return transactions, nil
}

// ListDebtorsByStore returns every user who currently owes storeID money, most recently updated
// first.
func (repo *DebtRepository) ListDebtorsByStore(ctx context.Context, storeID string) ([]*domain.UserDebtWithUser, error) {
	var storeUUID pgtype.UUID
	if err := storeUUID.Scan(storeID); err != nil {
		return nil, fmt.Errorf("invalid storeID: %w", domain.ErrInvalidInput)
	}

	results, err := repo.queries.ListDebtorsByStore(ctx, storeUUID)
	if err != nil {
		return nil, fmt.Errorf("listing debtors: %w", err)
	}

	debtors := make([]*domain.UserDebtWithUser, 0, len(results))
	for _, result := range results {
		debtors = append(debtors, &domain.UserDebtWithUser{
			UserDebt: domain.UserDebt{
				ID:        result.ID.String(),
				UserID:    result.UserID.String(),
				StoreID:   result.StoreID.String(),
				Balance:   result.Balance,
				CreatedAt: result.CreatedAt.Time.UTC(),
				UpdatedAt: result.UpdatedAt.Time.UTC(),
			},
			Username:  result.Username,
			Email:     result.Email,
			FirstName: result.FirstName,
			LastName:  result.LastName,
		})
	}

	return debtors, nil
}

// ListMyDebts returns every store userID currently owes money to, most recently updated first.
func (repo *DebtRepository) ListMyDebts(ctx context.Context, userID string) ([]*domain.UserDebtWithStore, error) {
	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		return nil, fmt.Errorf("invalid userID: %w", domain.ErrInvalidInput)
	}

	results, err := repo.queries.ListMyDebts(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("listing my debts: %w", err)
	}

	debts := make([]*domain.UserDebtWithStore, 0, len(results))
	for _, result := range results {
		debts = append(debts, &domain.UserDebtWithStore{
			UserDebt: domain.UserDebt{
				ID:        result.ID.String(),
				UserID:    result.UserID.String(),
				StoreID:   result.StoreID.String(),
				Balance:   result.Balance,
				CreatedAt: result.CreatedAt.Time.UTC(),
				UpdatedAt: result.UpdatedAt.Time.UTC(),
			},
			StoreName: result.StoreName,
		})
	}

	return debts, nil
}
