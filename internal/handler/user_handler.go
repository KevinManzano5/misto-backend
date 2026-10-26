// Package handler contains handlers
package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/KevinManzano5/misto-backend/internal/domain"
	"github.com/KevinManzano5/misto-backend/internal/usecase"
	"github.com/KevinManzano5/misto-backend/internal/validator"
)

// UserHandler struct
type UserHandler struct {
	usecase      *usecase.UserUseCase
	storeUsecase *usecase.StoreUseCase
	validator    *validator.Validator
	logger       *slog.Logger
}

// NewUserHandler constructor
func NewUserHandler(usecase *usecase.UserUseCase, storeUsecase *usecase.StoreUseCase, validator *validator.Validator, logger *slog.Logger) *UserHandler {
	return &UserHandler{
		usecase:      usecase,
		storeUsecase: storeUsecase,
		validator:    validator,
		logger:       logger,
	}
}

// userResponse struct
type userResponse struct {
	ID        string  `json:"id"`
	Username  string  `json:"username"`
	Email     string  `json:"email"`
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	HasStore  bool    `json:"has_store"`
	StoreID   *string `json:"store_id,omitempty"`
}

// userStoreInfo resolves whether the user owns a store and, if so, its id. A user without a
// store is an expected case (domain.ErrNotFound), not a failure of the request.
func (handler *UserHandler) userStoreInfo(ctx context.Context, userID string) (bool, *string, error) {
	store, err := handler.storeUsecase.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return false, nil, nil
		}

		return false, nil, err
	}

	return true, &store.ID, nil
}

// Get method
//
// @Summary Get User
// @Description Get an user by id
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Success 201 {object} userResponse
// @Failure 404 {object} ProblemDetails
// @Router /users/{id} [get]
func (handler *UserHandler) Get(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")

	user, err := handler.usecase.GetUser(request.Context(), id)

	if err != nil {
		RespondError(writer, request, handler.logger, err)

		return
	}

	hasStore, storeID, err := handler.userStoreInfo(request.Context(), user.ID)
	if err != nil {
		RespondError(writer, request, handler.logger, err)

		return
	}

	if err := RespondJSON(writer, http.StatusOK, userResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		HasStore:  hasStore,
		StoreID:   storeID,
	}); err != nil {
		handler.logger.Error("error trying to write JSON response",
			"error", err,
			"method", request.Method,
			"path", request.URL.Path)
	}
}
