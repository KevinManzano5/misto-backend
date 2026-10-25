package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
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

type upsertStoreScheduleRequest struct {
	OpenTime  string `json:"open_time" validate:"omitempty"`
	CloseTime string `json:"close_time" validate:"omitempty"`
	IsClosed  bool   `json:"is_closed"`
}

type storeScheduleResponse struct {
	ID        string    `json:"id"`
	StoreID   string    `json:"store_id"`
	DayOfWeek int32     `json:"day_of_week"`
	OpenTime  string    `json:"open_time"`
	CloseTime string    `json:"close_time"`
	IsClosed  bool      `json:"is_closed"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toStoreScheduleResponse(schedule *domain.StoreSchedule) storeScheduleResponse {
	return storeScheduleResponse{
		ID:        schedule.ID,
		StoreID:   schedule.StoreID,
		DayOfWeek: schedule.DayOfWeek,
		OpenTime:  schedule.OpenTime,
		CloseTime: schedule.CloseTime,
		IsClosed:  schedule.IsClosed,
		CreatedAt: schedule.CreatedAt,
		UpdatedAt: schedule.UpdatedAt,
	}
}

// parseDayOfWeek converts the {day} path value to a valid day-of-week (0=Sunday..6=Saturday).
func parseDayOfWeek(value string) (int32, error) {
	day, err := strconv.Atoi(value)
	if err != nil || day < 0 || day > 6 {
		return 0, domain.ErrInvalidInput
	}

	return int32(day), nil
}

type createStoreServiceRequest struct {
	Name            string  `json:"name" validate:"required,min=2,max=100"`
	Description     string  `json:"description"`
	DurationMinutes int32   `json:"duration_minutes" validate:"required,min=1"`
	Price           float64 `json:"price" validate:"min=0"`
}

type updateStoreServiceRequest struct {
	Name            *string  `json:"name" validate:"omitempty,min=2,max=100"`
	Description     *string  `json:"description"`
	DurationMinutes *int32   `json:"duration_minutes" validate:"omitempty,min=1"`
	Price           *float64 `json:"price" validate:"omitempty,min=0"`
	IsActive        *bool    `json:"is_active"`
}

type storeServiceResponse struct {
	ID              string    `json:"id"`
	StoreID         string    `json:"store_id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	DurationMinutes int32     `json:"duration_minutes"`
	Price           float64   `json:"price"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// StoreServiceHandler struct
type StoreServiceHandler struct {
	usecase    *usecase.StoreServiceUseCase
	appUsecase *usecase.AppointmentUseCase
	validator  *validator.Validator
	logger     *slog.Logger
}

// NewStoreServiceHandler constructor
func NewStoreServiceHandler(usecase *usecase.StoreServiceUseCase, appUsecase *usecase.AppointmentUseCase, validator *validator.Validator, logger *slog.Logger) *StoreServiceHandler {
	return &StoreServiceHandler{
		usecase:    usecase,
		appUsecase: appUsecase,
		validator:  validator,
		logger:     logger,
	}
}

func toStoreServiceResponse(service *domain.StoreService) storeServiceResponse {
	return storeServiceResponse{
		ID:              service.ID,
		StoreID:         service.StoreID,
		Name:            service.Name,
		Description:     service.Description,
		DurationMinutes: service.DurationMinutes,
		Price:           service.Price,
		IsActive:        service.IsActive,
		CreatedAt:       service.CreatedAt,
		UpdatedAt:       service.UpdatedAt,
	}
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
func (handler *StoreHandler) GetStoreByID(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")

	store, err := handler.usecase.GetByID(request.Context(), id)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusOK, toStoreResponse(store)); err != nil {
		handler.logger.Error(
			"error trying to write JSON response",
			"error", err,
			"method", request.Method,
			"path", request.URL.Path,
		)
	}
}

// GetStoreByUserID endpoint
func (handler *StoreHandler) GetStoreByUserID(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")

	store, err := handler.usecase.GetByUserID(request.Context(), id)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusOK, toStoreResponse(store)); err != nil {
		handler.logger.Error(
			"error trying to write JSON response",
			"error", err,
			"method", request.Method,
			"path", request.URL.Path,
		)
	}
}

// UpdateStore endpoint
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
func (handler *StoreHandler) DeleteStore(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")

	err := handler.usecase.Delete(request.Context(), id)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusOK, "store deleted"); err != nil {
		handler.logger.Error(
			"error trying to write JSON response",
			"error", err,
			"method", request.Method,
			"path", request.URL.Path,
		)
	}
}

// ListStores endpoint
func (handler *StoreHandler) ListStores(writer http.ResponseWriter, request *http.Request) {
	stores, err := handler.usecase.List(request.Context())
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	response := make([]storeResponse, 0)
	for _, store := range stores {
		response = append(response, toStoreResponse(store))
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

// UpsertSchedule endpoint (e.g., PUT /stores/{storeId}/schedules/{day})
func (handler *StoreHandler) UpsertSchedule(writer http.ResponseWriter, request *http.Request) {
	storeID := request.PathValue("storeId")

	dayOfWeek, err := parseDayOfWeek(request.PathValue("day"))
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

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

	var req upsertStoreScheduleRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		RespondError(writer, request, handler.logger, domain.ErrInvalidInput)
		return
	}

	if errs := handler.validator.Validate(req); errs != nil {
		if err := RespondValidationError(writer, errs); err != nil {
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

	schedule, err := handler.usecase.UpsertSchedule(request.Context(), storeID, dayOfWeek, req.OpenTime, req.CloseTime, req.IsClosed)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusOK, toStoreScheduleResponse(schedule)); err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"error trying to create response JSON",
			"error", err,
			"method", request.Method,
			"path", request.URL.Path,
		)
	}
}

// GetScheduleByDay endpoint (e.g., GET /stores/{storeId}/schedules/{day})
func (handler *StoreHandler) GetScheduleByDay(writer http.ResponseWriter, request *http.Request) {
	storeID := request.PathValue("storeId")

	dayOfWeek, err := parseDayOfWeek(request.PathValue("day"))
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	schedule, err := handler.usecase.GetScheduleByDay(request.Context(), storeID, dayOfWeek)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusOK, toStoreScheduleResponse(schedule)); err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"error trying to write JSON response",
			"error", err,
			"method", request.Method,
			"path", request.URL.Path,
		)
	}
}

// ListSchedules endpoint (e.g., GET /stores/{storeId}/schedules)
func (handler *StoreHandler) ListSchedules(writer http.ResponseWriter, request *http.Request) {
	storeID := request.PathValue("storeId")

	schedules, err := handler.usecase.ListSchedules(request.Context(), storeID)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	response := make([]storeScheduleResponse, 0)
	for _, schedule := range schedules {
		response = append(response, toStoreScheduleResponse(schedule))
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

// Create endpoint (e.g., POST /stores/{storeId}/services)
func (handler *StoreServiceHandler) Create(writer http.ResponseWriter, request *http.Request) {
	storeID := request.PathValue("storeId")

	if request.Body == nil {
		RespondError(writer, request, handler.logger, domain.ErrInvalidInput)
		return
	}
	defer func() {
		_ = request.Body.Close()
	}()

	var req createStoreServiceRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		RespondError(writer, request, handler.logger, domain.ErrInvalidInput)
		return
	}

	if errors := handler.validator.Validate(req); errors != nil {
		_ = RespondValidationError(writer, errors)
		return
	}

	service, err := handler.usecase.Create(request.Context(), storeID, req.Name, req.Description, req.DurationMinutes, req.Price, true)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusCreated, toStoreServiceResponse(service)); err != nil {
		handler.logger.ErrorContext(request.Context(), "failed to write response", "error", err)
	}
}

// GetByID endpoint (e.g., GET /services/{id})
func (handler *StoreServiceHandler) GetByID(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")

	service, err := handler.usecase.GetByID(request.Context(), id)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusOK, toStoreServiceResponse(service)); err != nil {
		handler.logger.ErrorContext(request.Context(), "failed to write response", "error", err)
	}
}

// ListByStoreID endpoint (e.g., GET /stores/{storeId}/services)
func (handler *StoreServiceHandler) ListByStoreID(writer http.ResponseWriter, request *http.Request) {
	storeID := request.PathValue("storeId")

	services, err := handler.usecase.ListByStoreID(request.Context(), storeID)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	response := make([]storeServiceResponse, 0)
	for _, svc := range services {
		response = append(response, toStoreServiceResponse(svc))
	}

	if err := RespondJSON(writer, http.StatusOK, response); err != nil {
		handler.logger.ErrorContext(request.Context(), "failed to write response", "error", err)
	}
}

// Update endpoint (e.g., PATCH /services/{id})
func (handler *StoreServiceHandler) Update(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")

	if request.Body == nil {
		RespondError(writer, request, handler.logger, domain.ErrInvalidInput)
		return
	}
	defer func() {
		_ = request.Body.Close()
	}()

	var req updateStoreServiceRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		RespondError(writer, request, handler.logger, domain.ErrInvalidInput)
		return
	}

	if req.Name == nil && req.Description == nil && req.DurationMinutes == nil && req.Price == nil && req.IsActive == nil {
		RespondError(writer, request, handler.logger, domain.ErrInvalidInput)
		return
	}

	if validationErrors := handler.validator.Validate(req); validationErrors != nil {
		_ = RespondValidationError(writer, validationErrors)
		return
	}

	service, err := handler.usecase.Update(request.Context(), id, req.Name, req.Description, req.DurationMinutes, req.Price, req.IsActive)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusOK, toStoreServiceResponse(service)); err != nil {
		handler.logger.ErrorContext(request.Context(), "failed to write response", "error", err)
	}
}

// Delete endpoint (e.g., DELETE /services/{id})
func (handler *StoreServiceHandler) Delete(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")

	if err := handler.usecase.Delete(request.Context(), id); err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusOK, "store service deleted"); err != nil {
		handler.logger.ErrorContext(request.Context(), "failed to write response", "error", err)
	}
}

// GetAvailability endpoint (e.g., GET /stores/{storeId}/availability)
func (handler *StoreServiceHandler) GetAvailability(writer http.ResponseWriter, request *http.Request) {
	storeID := request.PathValue("storeId")
	serviceID := request.URL.Query().Get("serviceId")
	dateStr := request.URL.Query().Get("date")

	if serviceID == "" || dateStr == "" {
		RespondError(writer, request, handler.logger, domain.ErrInvalidInput)
		return
	}

	// 1. Obtain the service duration
	service, err := handler.usecase.GetByID(request.Context(), serviceID)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	// 2. Calculate available slots based on duration and existing appointments
	slots, err := handler.appUsecase.GetAvailableSlots(request.Context(), storeID, service.DurationMinutes, dateStr)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	// We explicitly return an empty array if there are no slots, in order to avoid a 'null' value in the frontend JSON.
	if slots == nil {
		slots = make([]string, 0)
	}

	if err := RespondJSON(writer, http.StatusOK, slots); err != nil {
		handler.logger.ErrorContext(request.Context(), "failed to write response", "error", err)
	}
}
