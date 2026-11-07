// Package handler contains handlers
package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

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

// adminUserResponse struct carries the fields an admin needs to manage a user, beyond what the
// public userResponse exposes.
type adminUserResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	IsActive  bool      `json:"is_active"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// listUsersResponse struct
type listUsersResponse struct {
	Users      []adminUserResponse `json:"users"`
	TotalCount int64               `json:"total_count"`
	Page       int32               `json:"page"`
	PageSize   int32               `json:"page_size"`
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

// ListUsers endpoint (e.g., GET /users)
//
// @Summary     List all users
// @Description Paginated list of every system user, newest first. Admin-only.
// @Tags        users
// @Produce     json
// @Param       page      query int false "Page number (default 1)"
// @Param       page_size query int false "Page size (default 20, max 100)"
// @Success     200 {object} listUsersResponse
// @Failure     403 {object} ProblemDetails
// @Router      /users [get]
func (handler *UserHandler) ListUsers(writer http.ResponseWriter, request *http.Request) {
	page, pageSize := paginationParams(request, 1, 20)

	users, total, err := handler.usecase.ListUsers(request.Context(), page, pageSize)

	if err != nil {
		RespondError(writer, request, handler.logger, err)

		return
	}

	response := listUsersResponse{
		Users:      make([]adminUserResponse, 0, len(users)),
		TotalCount: total,
		Page:       page,
		PageSize:   pageSize,
	}

	for _, user := range users {
		response.Users = append(response.Users, adminUserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			IsActive:  user.IsActive,
			Role:      user.Role,
			CreatedAt: user.CreatedAt,
		})
	}

	if err := RespondJSON(writer, http.StatusOK, response); err != nil {
		handler.logger.Error("error trying to write JSON response",
			"error", err,
			"method", request.Method,
			"path", request.URL.Path)
	}
}
