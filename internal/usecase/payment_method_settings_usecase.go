package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/KevinManzano5/misto-backend/internal/domain"
)

// PaymentMethodSettingsRepository interface
type PaymentMethodSettingsRepository interface {
	GetPaymentMethodSettings(ctx context.Context) (*domain.PaymentMethodSettings, error)
	UpdatePaymentMethodSettings(ctx context.Context, cashOnSiteEnabled, cardAdvanceEnabled *bool) (*domain.PaymentMethodSettings, error)
}

// PaymentMethodSettingsUseCase struct
type PaymentMethodSettingsUseCase struct {
	repo PaymentMethodSettingsRepository
	log  *slog.Logger
}

// NewPaymentMethodSettingsUseCase constructor
func NewPaymentMethodSettingsUseCase(repo PaymentMethodSettingsRepository, log *slog.Logger) *PaymentMethodSettingsUseCase {
	return &PaymentMethodSettingsUseCase{
		repo: repo,
		log:  log,
	}
}

// Get returns the current platform-wide payment method settings.
func (usecase *PaymentMethodSettingsUseCase) Get(ctx context.Context) (*domain.PaymentMethodSettings, error) {
	settings, err := usecase.repo.GetPaymentMethodSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting payment method settings: %w", err)
	}

	return settings, nil
}

// Update sets the platform-wide payment method settings. Admin-only: authorization is enforced
// by the handler/route, not here. A nil field leaves that method's current setting unchanged.
func (usecase *PaymentMethodSettingsUseCase) Update(ctx context.Context, cashOnSiteEnabled, cardAdvanceEnabled *bool) (*domain.PaymentMethodSettings, error) {
	settings, err := usecase.repo.UpdatePaymentMethodSettings(ctx, cashOnSiteEnabled, cardAdvanceEnabled)
	if err != nil {
		return nil, fmt.Errorf("updating payment method settings: %w", err)
	}

	usecase.log.InfoContext(ctx, "payment method settings updated",
		slog.Bool("cash_on_site_enabled", settings.CashOnSiteEnabled),
		slog.Bool("card_advance_enabled", settings.CardAdvanceEnabled),
	)

	return settings, nil
}
