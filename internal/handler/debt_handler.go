// Package handler contains handlers
package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/KevinManzano5/misto-backend/internal/domain"
	"github.com/KevinManzano5/misto-backend/internal/middlewares"
	"github.com/KevinManzano5/misto-backend/internal/usecase"
	"github.com/KevinManzano5/misto-backend/internal/validator"
)

// DebtHandler struct
type DebtHandler struct {
	usecase   *usecase.DebtUseCase
	validator *validator.Validator
	logger    *slog.Logger
}

// NewDebtHandler constructor
func NewDebtHandler(usecase *usecase.DebtUseCase, validator *validator.Validator, logger *slog.Logger) *DebtHandler {
	return &DebtHandler{
		usecase:   usecase,
		validator: validator,
		logger:    logger,
	}
}

// debtResponse struct
type debtResponse struct {
	UserID    string    `json:"user_id"`
	StoreID   string    `json:"store_id"`
	Balance   int64     `json:"balance"`
	UpdatedAt time.Time `json:"updated_at"`
}

// debtorResponse struct
type debtorResponse struct {
	debtResponse
	Username  string `json:"username"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// myDebtResponse struct
type myDebtResponse struct {
	debtResponse
	StoreName string `json:"store_name"`
}

// debtTransactionResponse struct
type debtTransactionResponse struct {
	ID            string    `json:"id"`
	AppointmentID string    `json:"appointment_id,omitempty"`
	Amount        int64     `json:"amount"`
	Type          string    `json:"type"`
	CreatedAt     time.Time `json:"created_at"`
}

// buildDebtResponse function
func buildDebtResponse(debt *domain.UserDebt) debtResponse {
	return debtResponse{
		UserID:    debt.UserID,
		StoreID:   debt.StoreID,
		Balance:   debt.Balance,
		UpdatedAt: debt.UpdatedAt,
	}
}

// Get endpoint (GET /api/v1/stores/{storeId}/debts/{userId}). Returns a client's balance owed
// to the store. Only the store owner or the debtor themself may call this.
func (handler *DebtHandler) Get(writer http.ResponseWriter, request *http.Request) {
	storeID := request.PathValue("storeId")
	userID := request.PathValue("userId")

	debt, err := handler.usecase.GetDebt(request.Context(), storeID, middlewares.GetUserID(request), userID)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusOK, buildDebtResponse(debt)); err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"error trying to write JSON response",
			"error", err,
			"method", request.Method,
			"path", request.URL.Path,
		)
	}
}

// ListTransactions endpoint (GET /api/v1/stores/{storeId}/debts/{userId}/transactions). Returns
// the full debt history for a client at a store. Only the store owner or the debtor themself
// may call this.
func (handler *DebtHandler) ListTransactions(writer http.ResponseWriter, request *http.Request) {
	storeID := request.PathValue("storeId")
	userID := request.PathValue("userId")

	transactions, err := handler.usecase.ListTransactions(request.Context(), storeID, middlewares.GetUserID(request), userID)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	response := make([]debtTransactionResponse, 0, len(transactions))
	for _, transaction := range transactions {
		response = append(response, debtTransactionResponse{
			ID:            transaction.ID,
			AppointmentID: transaction.AppointmentID,
			Amount:        transaction.Amount,
			Type:          transaction.Type,
			CreatedAt:     transaction.CreatedAt,
		})
	}

	if err := RespondJSON(writer, http.StatusOK, response); err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"error trying to write JSON response",
			"error", err,
			"method", request.Method,
			"path", request.URL.Path,
		)
	}
}

// ListDebtorsByStore endpoint (GET /api/v1/stores/{storeId}/debts). Only the store owner may
// view every client who currently owes it money.
func (handler *DebtHandler) ListDebtorsByStore(writer http.ResponseWriter, request *http.Request) {
	storeID := request.PathValue("storeId")

	debtors, err := handler.usecase.ListDebtorsByStore(request.Context(), storeID, middlewares.GetUserID(request))
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	response := make([]debtorResponse, 0, len(debtors))
	for _, debtor := range debtors {
		response = append(response, debtorResponse{
			debtResponse: buildDebtResponse(&debtor.UserDebt),
			Username:     debtor.Username,
			Email:        debtor.Email,
			FirstName:    debtor.FirstName,
			LastName:     debtor.LastName,
		})
	}

	if err := RespondJSON(writer, http.StatusOK, response); err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"error trying to write JSON response",
			"error", err,
			"method", request.Method,
			"path", request.URL.Path,
		)
	}
}

// ListMyDebts endpoint (GET /api/v1/me/debts). Returns every store the authenticated client
// currently owes money to.
func (handler *DebtHandler) ListMyDebts(writer http.ResponseWriter, request *http.Request) {
	debts, err := handler.usecase.ListMyDebts(request.Context(), middlewares.GetUserID(request))
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	response := make([]myDebtResponse, 0, len(debts))
	for _, debt := range debts {
		response = append(response, myDebtResponse{
			debtResponse: buildDebtResponse(&debt.UserDebt),
			StoreName:    debt.StoreName,
		})
	}

	if err := RespondJSON(writer, http.StatusOK, response); err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"error trying to write JSON response",
			"error", err,
			"method", request.Method,
			"path", request.URL.Path,
		)
	}
}
