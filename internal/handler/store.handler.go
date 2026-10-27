package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	Name                string `json:"name" validate:"required,min=2,max=30"`
	Description         string `json:"description" validate:"required"`
	AppointmentDuration int32  `json:"appointment_duration" validate:"omitempty,min=1"`
	// Timezone is the store's IANA timezone (e.g. "America/Bogota"). Defaults to "UTC" when omitted.
	Timezone string `json:"timezone" validate:"omitempty"`
	// AutoConfirmAppointments controls whether new appointments are confirmed immediately or
	// left pending for the owner to approve/reject. Defaults to true (auto-confirm) when omitted.
	AutoConfirmAppointments *bool `json:"auto_confirm_appointments"`
	// Email is the store's contact email address.
	Email string `json:"email" validate:"omitempty,email"`
	// Address is the store's physical address.
	Address string `json:"address" validate:"omitempty"`
	// AllowSameDayAppointments controls whether a client can book an appointment on the same
	// calendar day it is requested. Defaults to true (allowed) when omitted.
	AllowSameDayAppointments *bool `json:"allow_same_day_appointments"`
}

type updateStoreRequest struct {
	Name                     *string `json:"name" validate:"omitempty,min=2,max=30"`
	Description              *string `json:"description"`
	IsActive                 *bool   `json:"is_active"`
	AppointmentDuration      *int32  `json:"appointment_duration" validate:"omitempty,min=1"`
	Timezone                 *string `json:"timezone" validate:"omitempty"`
	AutoConfirmAppointments  *bool   `json:"auto_confirm_appointments"`
	Email                    *string `json:"email" validate:"omitempty,email"`
	Address                  *string `json:"address"`
	AllowSameDayAppointments *bool   `json:"allow_same_day_appointments"`
}

type storeResponse struct {
	ID                       string                  `json:"id"`
	UserID                   string                  `json:"user_id"`
	Name                     string                  `json:"name"`
	Description              string                  `json:"description"`
	IsActive                 bool                    `json:"is_active"`
	AppointmentDuration      int32                   `json:"appointment_duration"`
	Timezone                 string                  `json:"timezone"`
	AutoConfirmAppointments  bool                    `json:"auto_confirm_appointments"`
	Email                    string                  `json:"email"`
	Address                  string                  `json:"address"`
	AllowSameDayAppointments bool                    `json:"allow_same_day_appointments"`
	Schedules                []storeScheduleResponse `json:"schedules"`
	CreatedAt                time.Time               `json:"created_at"`
	UpdatedAt                time.Time               `json:"updated_at"`
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

// buildStoreResponse composes the full store response, embedding its weekly schedules so
// callers don't need a separate request to see a store's opening hours.
func (handler *StoreHandler) buildStoreResponse(ctx context.Context, store *domain.Store) (storeResponse, error) {
	schedules, err := handler.usecase.ListSchedules(ctx, store.ID)
	if err != nil {
		return storeResponse{}, fmt.Errorf("listing store schedules: %w", err)
	}

	scheduleResponses := make([]storeScheduleResponse, 0, len(schedules))
	for _, schedule := range schedules {
		scheduleResponses = append(scheduleResponses, toStoreScheduleResponse(schedule))
	}

	return storeResponse{
		ID:                       store.ID,
		UserID:                   store.UserID,
		Name:                     store.Name,
		Description:              store.Description,
		IsActive:                 store.IsActive,
		AppointmentDuration:      store.AppointmentDuration,
		Timezone:                 store.Timezone,
		AutoConfirmAppointments:  store.AutoConfirmAppointments,
		Email:                    store.Email,
		Address:                  store.Address,
		AllowSameDayAppointments: store.AllowSameDayAppointments,
		Schedules:                scheduleResponses,
		CreatedAt:                store.CreatedAt,
		UpdatedAt:                store.UpdatedAt,
	}, nil
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

	autoConfirmAppointments := true
	if req.AutoConfirmAppointments != nil {
		autoConfirmAppointments = *req.AutoConfirmAppointments
	}

	allowSameDayAppointments := true
	if req.AllowSameDayAppointments != nil {
		allowSameDayAppointments = *req.AllowSameDayAppointments
	}

	store, err := handler.usecase.Create(request.Context(), userID, req.Name, req.Description, true, req.AppointmentDuration, req.Timezone, autoConfirmAppointments, req.Email, req.Address, allowSameDayAppointments)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	response, err := handler.buildStoreResponse(request.Context(), store)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusCreated, response); err != nil {
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

	response, err := handler.buildStoreResponse(request.Context(), store)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusOK, response); err != nil {
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

	response, err := handler.buildStoreResponse(request.Context(), store)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusOK, response); err != nil {
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

	if req.Name == nil && req.Description == nil && req.IsActive == nil && req.AppointmentDuration == nil && req.Timezone == nil && req.AutoConfirmAppointments == nil && req.Email == nil && req.Address == nil && req.AllowSameDayAppointments == nil {
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

	store, err := handler.usecase.Update(request.Context(), id, req.Name, req.Description, req.IsActive, req.AppointmentDuration, req.Timezone, req.AutoConfirmAppointments, req.Email, req.Address, req.AllowSameDayAppointments)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	response, err := handler.buildStoreResponse(request.Context(), store)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusOK, response); err != nil {
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

	response := make([]storeResponse, 0, len(stores))
	for _, store := range stores {
		storeWithSchedules, err := handler.buildStoreResponse(request.Context(), store)
		if err != nil {
			RespondError(writer, request, handler.logger, err)
			return
		}
		response = append(response, storeWithSchedules)
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
