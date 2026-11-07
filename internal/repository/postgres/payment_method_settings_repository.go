package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/KevinManzano5/misto-backend/internal/domain"
	postgres "github.com/KevinManzano5/misto-backend/internal/repository/postgres/generated"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PaymentMethodSettingsRepository struct
type PaymentMethodSettingsRepository struct {
	pool    *pgxpool.Pool
	queries *postgres.Queries
}

// toDomainPaymentMethodSettings function
func toDomainPaymentMethodSettings(settings postgres.PaymentMethodSetting) *domain.PaymentMethodSettings {
	return &domain.PaymentMethodSettings{
		CashOnSiteEnabled:  settings.CashOnSiteEnabled,
		CardAdvanceEnabled: settings.CardAdvanceEnabled,
		UpdatedAt:          settings.UpdatedAt.Time.UTC(),
	}
}

// NewPaymentMethodSettingsRepository function
func NewPaymentMethodSettingsRepository(pool *pgxpool.Pool) *PaymentMethodSettingsRepository {
	return &PaymentMethodSettingsRepository{
		pool:    pool,
		queries: postgres.New(pool),
	}
}

// GetPaymentMethodSettings method returns the single, platform-wide payment method settings row.
func (repo *PaymentMethodSettingsRepository) GetPaymentMethodSettings(ctx context.Context) (*domain.PaymentMethodSettings, error) {
	result, err := repo.queries.GetPaymentMethodSettings(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("payment method settings: %w", domain.ErrNotFound)
		}

		return nil, fmt.Errorf("getting payment method settings: %w", err)
	}

	return toDomainPaymentMethodSettings(result), nil
}

// UpdatePaymentMethodSettings method partially updates the platform-wide payment method
// settings. A nil field leaves that method's current setting unchanged.
func (repo *PaymentMethodSettingsRepository) UpdatePaymentMethodSettings(ctx context.Context, cashOnSiteEnabled, cardAdvanceEnabled *bool) (*domain.PaymentMethodSettings, error) {
	result, err := repo.queries.UpdatePaymentMethodSettings(ctx, postgres.UpdatePaymentMethodSettingsParams{
		CashOnSiteEnabled:  cashOnSiteEnabled,
		CardAdvanceEnabled: cardAdvanceEnabled,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("payment method settings: %w", domain.ErrNotFound)
		}

		return nil, fmt.Errorf("updating payment method settings: %w", err)
	}

	return toDomainPaymentMethodSettings(result), nil
}
