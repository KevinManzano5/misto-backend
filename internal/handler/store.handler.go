package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/KevinManzano5/misto-backend/internal/domain"
	"github.com/KevinManzano5/misto-backend/internal/middlewares"
	"github.com/KevinManzano5/misto-backend/internal/usecase"
	"github.com/KevinManzano5/misto-backend/internal/validator"
)

// StoreHandler struct
type StoreHandler struct {
	usecase   *usecase.StoreUseCase
	validator *validator.Validator
	logger    *slog.Logger
}

// NewStoreHandler constructor
func NewStoreHandler(usecase *usecase.StoreUseCase, validator *validator.Validator, logger *slog.Logger) *StoreHandler {
	return &StoreHandler{
		usecase:   usecase,
		validator: validator,
		logger:    logger,
	}
}

type createStoreRequest struct {
	Name        string `json:"name" validate:"required,min=2,max=30"`
	Description string `json:"description" validate:"required"`
}

type updateStoreRequest struct {
	Name        *string `json:"name" validate:"omitempty,min=2,max=30"`
	Description *string `json:"description"`
	IsActive    *bool   `json:"is_active"`
}

type storeResponse struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toStoreResponse(store *domain.Store) storeResponse {
	return storeResponse{
		ID:          store.ID,
		UserID:      store.UserID,
		Name:        store.Name,
		Description: store.Description,
		IsActive:    store.IsActive,
		CreatedAt:   store.CreatedAt,
		UpdatedAt:   store.UpdatedAt,
	}
}

// Create endpoint
//
// @Summary     Register a new store
// @Description Creates a new store
// @Tags        store
// @Accept      json
// @Produce     json
// @Param       request body     registerRequest true "Register data"
// @Success     201     {object} tokenResponse
// @Failure     400     {object} ProblemDetails
// @Failure     409     {object} ProblemDetails
// @Router      /auth/register [post]
func (handler *StoreHandler) Create(writer http.ResponseWriter, request *http.Request) {
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

	var req createStoreRequest

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

	userID := middlewares.GetUserID(request)

	store, err := handler.usecase.Create(request.Context(), userID, req.Name, req.Description, true)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusCreated, toStoreResponse(store)); err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"error trying to create response JSON",
			"error", err,
			"method", request.Method,
			"path", request.URL.Path,
		)
	}
}

// GetStoreByID endpoint
//
// @Summary     Register a new store
// @Description Creates a new store
// @Tags        store
// @Accept      json
// @Produce     json
// @Param       request body     registerRequest true "Register data"
// @Success     201     {object} tokenResponse
// @Failure     400     {object} ProblemDetails
// @Failure     409     {object} ProblemDetails
// @Router      /auth/register [post]
func (handler *StoreHandler) GetStoreByID(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")

	store, err := handler.usecase.GetByID(request.Context(), id)

	if err != nil {
		RespondError(writer, request, handler.logger, err)

		return
	}

	if err := respondJSON(writer, http.StatusOK, toStoreResponse(store)); err != nil {
		handler.logger.Error(
			"error trying to write JSON response",
			"error", err,
			"method", request.Method,
			"path", request.URL.Path,
		)
	}
}

// GetStoreByUserID endpoint
//
// @Summary     Register a new store
// @Description Creates a new store
// @Tags        store
// @Accept      json
// @Produce     json
// @Param       request body     registerRequest true "Register data"
// @Success     201     {object} tokenResponse
// @Failure     400     {object} ProblemDetails
// @Failure     409     {object} ProblemDetails
// @Router      /auth/register [post]
func (handler *StoreHandler) GetStoreByUserID(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")

	store, err := handler.usecase.GetByUserID(request.Context(), id)

	if err != nil {
		RespondError(writer, request, handler.logger, err)

		return
	}

	if err := respondJSON(writer, http.StatusOK, toStoreResponse(store)); err != nil {
		handler.logger.Error(
			"error trying to write JSON response",
			"error", err,
			"method", request.Method,
			"path", request.URL.Path,
		)
	}
}

// UpdateStore endpoint
//
// @Summary     Register a new store
// @Description Creates a new store
// @Tags        store
// @Accept      json
// @Produce     json
// @Param       request body     registerRequest true "Register data"
// @Success     201     {object} tokenResponse
// @Failure     400     {object} ProblemDetails
// @Failure     409     {object} ProblemDetails
// @Router      /auth/register [post]
func (handler *StoreHandler) UpdateStore(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")

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

	var req updateStoreRequest

	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		RespondError(writer, request, handler.logger, domain.ErrInvalidInput)
		return
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		RespondError(writer, request, handler.logger, domain.ErrInvalidInput)
		return
	}

	if req.Name == nil && req.Description == nil && req.IsActive == nil {
		RespondError(writer, request, handler.logger, domain.ErrInvalidInput)
		return
	}

	if validationErrors := handler.validator.Validate(req); validationErrors != nil {
		if err := RespondValidationError(writer, validationErrors); err != nil {
			handler.logger.ErrorContext(
				request.Context(),
				"error trying to write validation response",
				"error", err,
				"method", request.Method,
				"path", request.URL.Path,
			)
		}
		return
	}

	store, err := handler.usecase.Update(request.Context(), id, req.Name, req.Description, req.IsActive)

	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusOK, toStoreResponse(store)); err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"error trying to create response JSON",
			"error", err,
			"method", request.Method,
			"path", request.URL.Path,
		)
	}
}

// DeleteStore endpoint
//
// @Summary     Register a new store
// @Description Creates a new store
// @Tags        store
// @Accept      json
// @Produce     json
// @Param       request body     registerRequest true "Register data"
// @Success     201     {object} tokenResponse
// @Failure     400     {object} ProblemDetails
// @Failure     409     {object} ProblemDetails
// @Router      /auth/register [post]
func (handler *StoreHandler) DeleteStore(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")

	err := handler.usecase.Delete(request.Context(), id)

	if err != nil {
		RespondError(writer, request, handler.logger, err)

		return
	}

	if err := respondJSON(writer, http.StatusOK, "store deleted"); err != nil {
		handler.logger.Error(
			"error trying to write JSON response",
			"error", err,
			"method", request.Method,
			"path", request.URL.Path,
		)
	}
}

// ListStores endpoint
//
// @Summary     List all active stores
// @Description Retrieves an array of all active stores in the system
// @Tags        store
// @Produce     json
// @Success     200     {array}  storeResponse
// @Failure     500     {object} ProblemDetails
// @Router      /stores [get]
func (handler *StoreHandler) ListStores(writer http.ResponseWriter, request *http.Request) {
	stores, err := handler.usecase.List(request.Context())

	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	// We initialize the slice as empty rather than nil to ensure that the JSON returns [] instead of null.
	response := make([]storeResponse, 0)
	for _, store := range stores {
		response = append(response, toStoreResponse(store))
	}

	// We use RespondJSON (the capitalized version you have in your Create/Update)
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
