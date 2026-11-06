package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/KevinManzano5/misto-backend/internal/domain"
	"github.com/KevinManzano5/misto-backend/internal/middlewares"
	"github.com/KevinManzano5/misto-backend/internal/usecase"
	"github.com/KevinManzano5/misto-backend/internal/validator"
)

// StoreServiceHandler struct
type StoreServiceHandler struct {
	usecase   *usecase.StoreServiceUseCase
	validator *validator.Validator
	logger    *slog.Logger
}

// NewStoreServiceHandler constructor
func NewStoreServiceHandler(usecase *usecase.StoreServiceUseCase, validator *validator.Validator, logger *slog.Logger) *StoreServiceHandler {
	return &StoreServiceHandler{
		usecase:   usecase,
		validator: validator,
		logger:    logger,
	}
}

// storeServiceResponse struct
type storeServiceResponse struct {
	ID              string    `json:"id"`
	StoreID         string    `json:"store_id"`
	CategoryID      string    `json:"category_id"`
	Name            string    `json:"name"`
	Description     string    `json:"description,omitempty"`
	DurationMinutes int32     `json:"duration_minutes"`
	Price           float64   `json:"price"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// toStoreServiceResponse function
func toStoreServiceResponse(service *domain.StoreService) storeServiceResponse {
	return storeServiceResponse{
		ID:              service.ID,
		StoreID:         service.StoreID,
		CategoryID:      service.CategoryID,
		Name:            service.Name,
		Description:     service.Description,
		DurationMinutes: service.DurationMinutes,
		Price:           service.Price,
		IsActive:        service.IsActive,
		CreatedAt:       service.CreatedAt,
		UpdatedAt:       service.UpdatedAt,
	}
}

// createStoreServiceRequest struct
type createStoreServiceRequest struct {
	CategoryID      string  `json:"category_id" validate:"required,uuid"`
	Name            string  `json:"name" validate:"required,min=2,max=100"`
	Description     string  `json:"description" validate:"omitempty,max=1000"`
	DurationMinutes int32   `json:"duration_minutes" validate:"required,min=1"`
	Price           float64 `json:"price" validate:"omitempty,min=0"`
}

// updateStoreServiceRequest struct
type updateStoreServiceRequest struct {
	CategoryID      *string  `json:"category_id" validate:"omitempty,uuid"`
	Name            *string  `json:"name" validate:"omitempty,min=2,max=100"`
	Description     *string  `json:"description" validate:"omitempty,max=1000"`
	DurationMinutes *int32   `json:"duration_minutes" validate:"omitempty,min=1"`
	Price           *float64 `json:"price" validate:"omitempty,min=0"`
	IsActive        *bool    `json:"is_active"`
}

// decodeJSONBody decodes a request body into dest, rejecting unknown fields and trailing data.
func decodeStoreServiceJSONBody(request *http.Request, dest any) error {
	if request.Body == nil {
		return domain.ErrInvalidInput
	}

	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dest); err != nil {
		return domain.ErrInvalidInput
	}

	if decoder.More() {
		return domain.ErrInvalidInput
	}

	return nil
}

// Create endpoint (POST /api/v1/stores/{storeId}/services). Creates a new service for a store;
// only the store's owner may call this, and the service's category must be one of the
// categories currently assigned to the store.
//
// @Summary Create a store service
// @Description Creates a service offered by a store, scoped to one of the store's assigned categories.
// @Tags store_services
// @Accept json
// @Produce json
// @Param storeId path string true "Store ID"
// @Param request body createStoreServiceRequest true "Service data"
// @Success 201 {object} storeServiceResponse
// @Router /stores/{storeId}/services [post]
func (handler *StoreServiceHandler) Create(writer http.ResponseWriter, request *http.Request) {
	storeID := request.PathValue("storeId")

	var req createStoreServiceRequest
	if err := decodeStoreServiceJSONBody(request, &req); err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if validationErrs := handler.validator.Validate(req); validationErrs != nil {
		if err := RespondValidationError(writer, validationErrs); err != nil {
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

	service, err := handler.usecase.Create(
		request.Context(),
		storeID,
		middlewares.GetUserID(request),
		req.CategoryID,
		req.Name,
		req.Description,
		req.DurationMinutes,
		req.Price,
	)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusCreated, toStoreServiceResponse(service)); err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"error trying to write JSON response",
			"error", err,
			"method", request.Method,
			"path", request.URL.Path,
		)
	}
}

// Update endpoint (PATCH /api/v1/services/{id}). Applies partial changes to a service; only the
// store's owner may call this.
//
// @Summary Update a store service
// @Description Applies partial changes to an existing store service.
// @Tags store_services
// @Accept json
// @Produce json
// @Param id path string true "Service ID"
// @Param request body updateStoreServiceRequest true "Fields to update"
// @Success 200 {object} storeServiceResponse
// @Router /services/{id} [patch]
func (handler *StoreServiceHandler) Update(writer http.ResponseWriter, request *http.Request) {
	serviceID := request.PathValue("id")

	var req updateStoreServiceRequest
	if err := decodeStoreServiceJSONBody(request, &req); err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if validationErrs := handler.validator.Validate(req); validationErrs != nil {
		if err := RespondValidationError(writer, validationErrs); err != nil {
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

	service, err := handler.usecase.Update(
		request.Context(),
		serviceID,
		middlewares.GetUserID(request),
		req.CategoryID,
		req.Name,
		req.Description,
		req.DurationMinutes,
		req.Price,
		req.IsActive,
	)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusOK, toStoreServiceResponse(service)); err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"error trying to write JSON response",
			"error", err,
			"method", request.Method,
			"path", request.URL.Path,
		)
	}
}

// Delete endpoint (DELETE /api/v1/services/{id}). Removes a service; only the store's owner may
// call this.
//
// @Summary Delete a store service
// @Description Deletes a store service.
// @Tags store_services
// @Produce json
// @Param id path string true "Service ID"
// @Success 200 {object} map[string]string
// @Router /services/{id} [delete]
func (handler *StoreServiceHandler) Delete(writer http.ResponseWriter, request *http.Request) {
	serviceID := request.PathValue("id")

	if err := handler.usecase.Delete(request.Context(), serviceID, middlewares.GetUserID(request)); err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusOK, "service deleted"); err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"error trying to write JSON response",
			"error", err,
			"method", request.Method,
			"path", request.URL.Path,
		)
	}
}

// ListByStore endpoint (GET /api/v1/stores/{storeId}/services). Returns the active services
// offered by a store. Public: no authorization check.
//
// @Summary List a store's services
// @Description Returns the active services offered by a store.
// @Tags store_services
// @Produce json
// @Param storeId path string true "Store ID"
// @Success 200 {array} storeServiceResponse
// @Router /stores/{storeId}/services [get]
func (handler *StoreServiceHandler) ListByStore(writer http.ResponseWriter, request *http.Request) {
	storeID := request.PathValue("storeId")

	services, err := handler.usecase.ListByStore(request.Context(), storeID)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	response := make([]storeServiceResponse, 0, len(services))
	for _, service := range services {
		response = append(response, toStoreServiceResponse(service))
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
