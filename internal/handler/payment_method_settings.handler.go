package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/KevinManzano5/misto-backend/internal/domain"
	"github.com/KevinManzano5/misto-backend/internal/usecase"
	"github.com/KevinManzano5/misto-backend/internal/validator"
)

// PaymentMethodSettingsHandler struct
type PaymentMethodSettingsHandler struct {
	usecase   *usecase.PaymentMethodSettingsUseCase
	validator *validator.Validator
	logger    *slog.Logger
}

// NewPaymentMethodSettingsHandler constructor
func NewPaymentMethodSettingsHandler(usecase *usecase.PaymentMethodSettingsUseCase, validator *validator.Validator, logger *slog.Logger) *PaymentMethodSettingsHandler {
	return &PaymentMethodSettingsHandler{
		usecase:   usecase,
		validator: validator,
		logger:    logger,
	}
}

// paymentMethodSettingsResponse struct
type paymentMethodSettingsResponse struct {
	CashOnSiteEnabled  bool      `json:"cash_on_site_enabled"`
	CardAdvanceEnabled bool      `json:"card_advance_enabled"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// updatePaymentMethodSettingsRequest struct. Both fields are pointers so a PATCH can toggle
// either/both independently; at least one must be present.
type updatePaymentMethodSettingsRequest struct {
	CashOnSiteEnabled  *bool `json:"cash_on_site_enabled"`
	CardAdvanceEnabled *bool `json:"card_advance_enabled"`
}

// buildPaymentMethodSettingsResponse function
func buildPaymentMethodSettingsResponse(settings *domain.PaymentMethodSettings) paymentMethodSettingsResponse {
	return paymentMethodSettingsResponse{
		CashOnSiteEnabled:  settings.CashOnSiteEnabled,
		CardAdvanceEnabled: settings.CardAdvanceEnabled,
		UpdatedAt:          settings.UpdatedAt,
	}
}

// Get endpoint (GET /api/v1/payment-methods). Admin-only: returns the platform-wide payment
// method settings.
func (handler *PaymentMethodSettingsHandler) Get(writer http.ResponseWriter, request *http.Request) {
	settings, err := handler.usecase.Get(request.Context())
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusOK, buildPaymentMethodSettingsResponse(settings)); err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"error trying to write JSON response",
			"error", err,
			"method", request.Method,
			"path", request.URL.Path,
		)
	}
}

// Update endpoint (PATCH /api/v1/payment-methods). Admin-only: toggles a payment method on or
// off for the entire platform, overriding any store's own per-store setting.
func (handler *PaymentMethodSettingsHandler) Update(writer http.ResponseWriter, request *http.Request) {
	if request.Body == nil {
		RespondError(writer, request, handler.logger, domain.ErrInvalidInput)
		return
	}
	defer func() {
		if err := request.Body.Close(); err != nil {
			handler.logger.ErrorContext(
				request.Context(),
				"failed to close request body",
				"error", err,
				"method", request.Method,
				"path", request.URL.Path,
			)
		}
	}()

	var req updatePaymentMethodSettingsRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		RespondError(writer, request, handler.logger, domain.ErrInvalidInput)
		return
	}

	if decoder.More() {
		RespondError(writer, request, handler.logger, domain.ErrInvalidInput)
		return
	}

	if req.CashOnSiteEnabled == nil && req.CardAdvanceEnabled == nil {
		RespondError(writer, request, handler.logger, fmt.Errorf("%w: at least one field must be provided", domain.ErrInvalidInput))
		return
	}

	settings, err := handler.usecase.Update(request.Context(), req.CashOnSiteEnabled, req.CardAdvanceEnabled)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusOK, buildPaymentMethodSettingsResponse(settings)); err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"error trying to create response JSON",
			"error", err,
			"method", request.Method,
			"path", request.URL.Path,
		)
	}
}
