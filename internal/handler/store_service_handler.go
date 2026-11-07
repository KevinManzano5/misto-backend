package handler

import (
	"context"
	"encoding/json"
	"fmt"
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
	ID                 string                         `json:"id"`
	StoreID            string                         `json:"store_id"`
	CategoryID         string                         `json:"category_id"`
	Name               string                         `json:"name"`
	Description        string                         `json:"description,omitempty"`
	DurationMinutes    int32                          `json:"duration_minutes"`
	Price              float64                        `json:"price"`
	IsActive           bool                           `json:"is_active"`
	MinLeadTimeMinutes int32                          `json:"min_lead_time_minutes"`
	Schedules          []storeServiceScheduleResponse `json:"schedules"`
	CreatedAt          time.Time                      `json:"created_at"`
	UpdatedAt          time.Time                      `json:"updated_at"`
}

// toStoreServiceResponse function
func toStoreServiceResponse(service *domain.StoreService) storeServiceResponse {
	return storeServiceResponse{
		ID:                 service.ID,
		StoreID:            service.StoreID,
		CategoryID:         service.CategoryID,
		Name:               service.Name,
		Description:        service.Description,
		DurationMinutes:    service.DurationMinutes,
		Price:              service.Price,
		IsActive:           service.IsActive,
		MinLeadTimeMinutes: service.MinLeadTimeMinutes,
		CreatedAt:          service.CreatedAt,
		UpdatedAt:          service.UpdatedAt,
	}
}

// storeServiceScheduleSegmentRequest struct
type storeServiceScheduleSegmentRequest struct {
	OpenTime  string `json:"open_time" validate:"required,hhmm"`
	CloseTime string `json:"close_time" validate:"required,hhmm"`
}

// upsertStoreServiceScheduleRequest struct
type upsertStoreServiceScheduleRequest struct {
	Segments []storeServiceScheduleSegmentRequest `json:"segments" validate:"required_if=IsClosed false,dive"`
	IsClosed bool                                 `json:"is_closed"`
}

// storeServiceScheduleSegmentResponse struct
type storeServiceScheduleSegmentResponse struct {
	ID        string `json:"id"`
	OpenTime  string `json:"open_time"`
	CloseTime string `json:"close_time"`
}

// storeServiceScheduleResponse struct
type storeServiceScheduleResponse struct {
	ID             string                                `json:"id"`
	StoreServiceID string                                `json:"store_service_id"`
	DayOfWeek      int32                                 `json:"day_of_week"`
	Segments       []storeServiceScheduleSegmentResponse `json:"segments"`
	IsClosed       bool                                  `json:"is_closed"`
	CreatedAt      time.Time                             `json:"created_at"`
	UpdatedAt      time.Time                             `json:"updated_at"`
}

// toStoreServiceScheduleResponse function
func toStoreServiceScheduleResponse(schedule *domain.StoreServiceSchedule) storeServiceScheduleResponse {
	segments := make([]storeServiceScheduleSegmentResponse, 0, len(schedule.Segments))
	for _, segment := range schedule.Segments {
		segments = append(segments, storeServiceScheduleSegmentResponse{
			ID:        segment.ID,
			OpenTime:  segment.OpenTime,
			CloseTime: segment.CloseTime,
		})
	}

	return storeServiceScheduleResponse{
		ID:             schedule.ID,
		StoreServiceID: schedule.StoreServiceID,
		DayOfWeek:      schedule.DayOfWeek,
		Segments:       segments,
		IsClosed:       schedule.IsClosed,
		CreatedAt:      schedule.CreatedAt,
		UpdatedAt:      schedule.UpdatedAt,
	}
}

// buildStoreServiceResponse composes a service's response, embedding its weekly schedule so
// callers don't need a separate request to see it.
func (handler *StoreServiceHandler) buildStoreServiceResponse(ctx context.Context, service *domain.StoreService) (storeServiceResponse, error) {
	schedules, err := handler.usecase.ListSchedules(ctx, service.ID)
	if err != nil {
		return storeServiceResponse{}, fmt.Errorf("listing store service schedules: %w", err)
	}

	scheduleResponses := make([]storeServiceScheduleResponse, 0, len(schedules))
	for _, schedule := range schedules {
		scheduleResponses = append(scheduleResponses, toStoreServiceScheduleResponse(schedule))
	}

	response := toStoreServiceResponse(service)
	response.Schedules = scheduleResponses

	return response, nil
}

// createStoreServiceRequest struct
type createStoreServiceRequest struct {
	CategoryID         string  `json:"category_id" validate:"required,uuid"`
	Name               string  `json:"name" validate:"required,min=2,max=100"`
	Description        string  `json:"description" validate:"omitempty,max=1000"`
	DurationMinutes    int32   `json:"duration_minutes" validate:"required,min=1"`
	Price              float64 `json:"price" validate:"omitempty,min=0"`
	MinLeadTimeMinutes int32   `json:"min_lead_time_minutes" validate:"omitempty,min=0"`
}

// updateStoreServiceRequest struct
type updateStoreServiceRequest struct {
	CategoryID         *string  `json:"category_id" validate:"omitempty,uuid"`
	Name               *string  `json:"name" validate:"omitempty,min=2,max=100"`
	Description        *string  `json:"description" validate:"omitempty,max=1000"`
	DurationMinutes    *int32   `json:"duration_minutes" validate:"omitempty,min=1"`
	Price              *float64 `json:"price" validate:"omitempty,min=0"`
	IsActive           *bool    `json:"is_active"`
	MinLeadTimeMinutes *int32   `json:"min_lead_time_minutes" validate:"omitempty,min=0"`
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
		req.MinLeadTimeMinutes,
	)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	response, err := handler.buildStoreServiceResponse(request.Context(), service)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusCreated, response); err != nil {
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
		req.MinLeadTimeMinutes,
	)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	response, err := handler.buildStoreServiceResponse(request.Context(), service)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
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
		serviceResponse, err := handler.buildStoreServiceResponse(request.Context(), service)
		if err != nil {
			RespondError(writer, request, handler.logger, err)
			return
		}
		response = append(response, serviceResponse)
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

// UpsertSchedule endpoint (PUT /api/v1/services/{serviceId}/schedules/{day}). Creates or updates
// a service's schedule for a specific day; only the store's owner may call this.
//
// @Summary Upsert a store service's schedule for a day
// @Description Creates or updates the segmented opening hours of a service for a specific day of the week.
// @Tags store_services
// @Accept json
// @Produce json
// @Param serviceId path string true "Service ID"
// @Param day path string true "Day of week (0=Sunday..6=Saturday)"
// @Param request body upsertStoreServiceScheduleRequest true "Schedule data"
// @Success 200 {object} storeServiceScheduleResponse
// @Router /services/{serviceId}/schedules/{day} [put]
func (handler *StoreServiceHandler) UpsertSchedule(writer http.ResponseWriter, request *http.Request) {
	serviceID := request.PathValue("serviceId")

	dayOfWeek, err := parseDayOfWeek(request.PathValue("day"))
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	var req upsertStoreServiceScheduleRequest
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

	segments := make([]domain.StoreServiceScheduleSegment, 0, len(req.Segments))
	for _, segment := range req.Segments {
		segments = append(segments, domain.StoreServiceScheduleSegment{OpenTime: segment.OpenTime, CloseTime: segment.CloseTime})
	}

	schedule, err := handler.usecase.UpsertSchedule(request.Context(), serviceID, middlewares.GetUserID(request), dayOfWeek, segments, req.IsClosed)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusOK, toStoreServiceScheduleResponse(schedule)); err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"error trying to write JSON response",
			"error", err,
			"method", request.Method,
			"path", request.URL.Path,
		)
	}
}
