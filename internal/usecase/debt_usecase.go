// Package usecase contains business logic
package usecase

import (
	"context"
	"fmt"

	"github.com/KevinManzano5/misto-backend/internal/domain"
)

// DebtRepository interface
type DebtRepository interface {
	GetUserDebt(ctx context.Context, userID, storeID string) (*domain.UserDebt, error)
	ListDebtTransactions(ctx context.Context, userID, storeID string) ([]*domain.DebtTransaction, error)
	ListDebtorsByStore(ctx context.Context, storeID string) ([]*domain.UserDebtWithUser, error)
	ListMyDebts(ctx context.Context, userID string) ([]*domain.UserDebtWithStore, error)
}

// DebtStoreRepository interface (narrowed to what DebtUseCase needs from a store)
type DebtStoreRepository interface {
	GetStoreByID(ctx context.Context, storeID string) (*domain.Store, error)
}

// DebtUseCase struct
type DebtUseCase struct {
	repo      DebtRepository
	storeRepo DebtStoreRepository
}

// NewDebtUseCase constructor
func NewDebtUseCase(repo DebtRepository, storeRepo DebtStoreRepository) *DebtUseCase {
	return &DebtUseCase{
		repo:      repo,
		storeRepo: storeRepo,
	}
}

// authorizeStoreOwnerOrDebtor loads storeID and returns it after checking that requesterUserID
// is either its owner or userID (the debtor). Anyone else gets domain.ErrForbidden.
func (usecase *DebtUseCase) authorizeStoreOwnerOrDebtor(ctx context.Context, storeID, requesterUserID, userID string) (*domain.Store, error) {
	store, err := usecase.storeRepo.GetStoreByID(ctx, storeID)
	if err != nil {
		return nil, fmt.Errorf("getting store: %w", err)
	}

	if requesterUserID != store.UserID && requesterUserID != userID {
		return nil, fmt.Errorf("%w: only the store owner or the debtor can view this debt", domain.ErrForbidden)
	}

	return store, nil
}

// GetDebt returns userID's balance owed to storeID. Only the store owner or the debtor
// themself may view it.
func (usecase *DebtUseCase) GetDebt(ctx context.Context, storeID, requesterUserID, userID string) (*domain.UserDebt, error) {
	if _, err := usecase.authorizeStoreOwnerOrDebtor(ctx, storeID, requesterUserID, userID); err != nil {
		return nil, err
	}

	debt, err := usecase.repo.GetUserDebt(ctx, userID, storeID)
	if err != nil {
		return nil, fmt.Errorf("getting user debt: %w", err)
	}

	return debt, nil
}

// ListTransactions returns the full debt history for userID at storeID, most recent first.
// Only the store owner or the debtor themself may view it.
func (usecase *DebtUseCase) ListTransactions(ctx context.Context, storeID, requesterUserID, userID string) ([]*domain.DebtTransaction, error) {
	if _, err := usecase.authorizeStoreOwnerOrDebtor(ctx, storeID, requesterUserID, userID); err != nil {
		return nil, err
	}

	transactions, err := usecase.repo.ListDebtTransactions(ctx, userID, storeID)
	if err != nil {
		return nil, fmt.Errorf("listing debt transactions: %w", err)
	}

	return transactions, nil
}

// ListDebtorsByStore returns every user who currently owes storeID money. Only the store
// owner may call this.
func (usecase *DebtUseCase) ListDebtorsByStore(ctx context.Context, storeID, requesterUserID string) ([]*domain.UserDebtWithUser, error) {
	store, err := usecase.storeRepo.GetStoreByID(ctx, storeID)
	if err != nil {
		return nil, fmt.Errorf("getting store: %w", err)
	}

	if requesterUserID != store.UserID {
		return nil, fmt.Errorf("%w: only the store owner can view its debtors", domain.ErrForbidden)
	}

	debtors, err := usecase.repo.ListDebtorsByStore(ctx, storeID)
	if err != nil {
		return nil, fmt.Errorf("listing debtors: %w", err)
	}

	return debtors, nil
}

// ListMyDebts returns every store userID currently owes money to.
func (usecase *DebtUseCase) ListMyDebts(ctx context.Context, userID string) ([]*domain.UserDebtWithStore, error) {
	debts, err := usecase.repo.ListMyDebts(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("listing my debts: %w", err)
	}

	return debts, nil
}
