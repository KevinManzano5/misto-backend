// Package handler contains handlers
package handler

import (
	"log/slog"
	"net/http"

	"github.com/KevinManzano5/misto-backend/internal/usecase"
	"github.com/KevinManzano5/misto-backend/internal/validator"
)

// UserHandler struct
type UserHandler struct {
	usecase   *usecase.UserUseCase
	validator *validator.Validator
	logger    *slog.Logger
}

// NewUserHandler constructor
func NewUserHandler(usecase *usecase.UserUseCase, validator *validator.Validator, logger *slog.Logger) *UserHandler {
	return &UserHandler{
		usecase:   usecase,
		validator: validator,
		logger:    logger,
	}
}

// userResponse struct
type userResponse struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
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

	if err := RespondJSON(writer, http.StatusOK, userResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
	}); err != nil {
		handler.logger.Error("error trying to write JSON response",
			"error", err,
			"method", request.Method,
			"path", request.URL.Path)
	}
}
