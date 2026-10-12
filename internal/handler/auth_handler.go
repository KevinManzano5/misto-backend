// Package handler contains handlers
package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/KevinManzano5/misto-backend/internal/domain"
	"github.com/KevinManzano5/misto-backend/internal/usecase"
	"github.com/KevinManzano5/misto-backend/internal/validator"
)

// AuthHandler struct
type AuthHandler struct {
	usecase   *usecase.AuthUseCase
	validator *validator.Validator
	logger    *slog.Logger
}

// NewAuthHandler func
func NewAuthHandler(usecase *usecase.AuthUseCase, validator *validator.Validator, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{
		usecase:   usecase,
		validator: validator,
		logger:    logger,
	}
}

// registerRequest struct
type signUpRequest struct {
	Username  string `json:"username" validate:"required,min=2,max=16"`
	Email     string `json:"email" validate:"required,email"`
	FirstName string `json:"first_name" validate:"required,min=2,max=100"`
	LastName  string `json:"last_name" validate:"required,min=2,max=100"`
	Password  string `json:"password" validate:"required,min=8,max=72"`
}

type signInRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// refreshRequest struct
type refreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// logoutRequest struct
type logoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// tokenResponse struct
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// toTokenResponse function
func toTokenResponse(tokens *usecase.AuthTokens) tokenResponse {
	return tokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(tokens.ExpiresIn.Seconds()),
	}
}

// SignUp endpoint
//
// @Summary     Register a new user
// @Description Creates a new user and return access tokens
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       request body     registerRequest true "Register data"
// @Success     201     {object} tokenResponse
// @Failure     400     {object} ProblemDetails
// @Failure     409     {object} ProblemDetails
// @Router      /auth/register [post]
func (handler *AuthHandler) SignUp(writer http.ResponseWriter, request *http.Request) {
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

	var req signUpRequest

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

	if errors := handler.validator.Validate(req); errors != nil {
		if err := RespondValidationError(writer, errors); err != nil {
			handler.logger.ErrorContext(
				request.Context(),
				"error trying to create validation response",
				"error", err,
				"method", request.Method,
				"path", request.URL.Path,
			)
		}
		return
	}

	tokens, err := handler.usecase.SignUp(request.Context(), req.Username, req.Email, req.FirstName, req.LastName, req.Password)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusCreated, toTokenResponse(tokens)); err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"error trying to create response JSON",
			"error", err,
			"method", request.Method,
			"path", request.URL.Path,
		)
	}
}

// SignIn endpoint
//
// @Summary     Sign In
// @Description Verifies credentials and returns access tokens
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       request body     loginRequest true "Credentials"
// @Success     200     {object} tokenResponse
// @Failure     401     {object} ProblemDetails
// @Router      /auth/login [post]
func (handler *AuthHandler) SignIn(writer http.ResponseWriter, request *http.Request) {
	var req signInRequest

	if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
		RespondError(writer, request, handler.logger, domain.ErrInvalidInput)
		return
	}

	if errors := handler.validator.Validate(req); errors != nil {
		if err := RespondValidationError(writer, errors); err != nil {
			handler.logger.ErrorContext(
				request.Context(),
				"error trying to create validation response",
				"error", err,
				"method", request.Method,
				"path", request.URL.Path,
			)
		}
		return
	}

	tokens, err := handler.usecase.SignIn(request.Context(), req.Email, req.Password)

	if err != nil {
		RespondError(writer, request, handler.logger, err)

		return
	}

	if err := RespondJSON(writer, http.StatusOK, toTokenResponse(tokens)); err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"error trying to create response JSON",
			"error", err,
			"method", request.Method,
			"path", request.URL.Path,
		)
	}
}

// Refresh endpoint
//
// @Summary     Refresh tokens
// @Description Refresh tokens using a valid refresh token
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       request body     refreshRequest true "Refresh token"
// @Success     200     {object} tokenResponse
// @Failure     401     {object} ProblemDetails
// @Router      /auth/refresh [post]
func (handler *AuthHandler) Refresh(writer http.ResponseWriter, request *http.Request) {
	var req refreshRequest

	if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
		RespondError(writer, request, handler.logger, domain.ErrInvalidInput)

		return
	}

	if errors := handler.validator.Validate(req); errors != nil {
		if err := RespondValidationError(writer, errors); err != nil {
			handler.logger.ErrorContext(
				request.Context(),
				"error trying to create validation response",
				"error", err,
				"method", request.Method,
				"path", request.URL.Path,
			)
		}
		return
	}

	tokens, err := handler.usecase.RefreshTokens(request.Context(), req.RefreshToken)

	if err != nil {
		RespondError(writer, request, handler.logger, err)

		return
	}

	if err := RespondJSON(writer, http.StatusOK, toTokenResponse(tokens)); err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"error trying to create response JSON",
			"error", err,
			"method", request.Method,
			"path", request.URL.Path,
		)
	}
}

// SignOut endpoint
//
// @Summary     Logout
// @Description Revoke user token
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       request body refreshRequest true "Refresh token to revoke"
// @Success     204     "Without content"
// @Router      /auth/logout [post]
func (handler *AuthHandler) SignOut(writer http.ResponseWriter, request *http.Request) {
	var req logoutRequest

	if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
		RespondError(writer, request, handler.logger, domain.ErrInvalidInput)

		return
	}

	if err := handler.usecase.SignOut(request.Context(), req.RefreshToken); err != nil {
		RespondError(writer, request, handler.logger, err)

		return
	}

	writer.WriteHeader(http.StatusNoContent)
}
