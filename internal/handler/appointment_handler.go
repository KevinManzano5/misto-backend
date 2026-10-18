// Package handler contains handlers
package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/KevinManzano5/misto-backend/internal/domain"
	"github.com/KevinManzano5/misto-backend/internal/usecase"
	"github.com/KevinManzano5/misto-backend/internal/validator"
)

// AppointmentHandler struct
type AppointmentHandler struct {
	usecase   *usecase.AppointmentUseCase
	validator *validator.Validator
	logger    *slog.Logger
}

// NewAppointmentHandler constructor
func NewAppointmentHandler(usecase *usecase.AppointmentUseCase, validator *validator.Validator, logger *slog.Logger) *AppointmentHandler {
	return &AppointmentHandler{
		usecase:   usecase,
		validator: validator,
		logger:    logger,
	}
}

// createAppointmentRequest struct
type createAppointmentRequest struct {
	StoreID         string    `json:"store_id" validate:"required,uuid"`
	ServiceID       string    `json:"service_id" validate:"required,uuid"`
	ClientID        string    `json:"client_id" validate:"required,uuid"`
	StartTime       time.Time `json:"start_time" validate:"required"`
	DurationMinutes int       `json:"duration_minutes" validate:"required,gt=0"`
}

// appointmentResponse struct
type appointmentResponse struct {
	ID        string    `json:"id"`
	StoreID   string    `json:"store_id"`
	ServiceID string    `json:"service_id"`
	UserID    string    `json:"user_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Status    string    `json:"status"`
}

// Create method
//
// @Summary Create Appointment
// @Description Schedule a new appointment for a store service
// @Tags appointments
// @Accept json
// @Produce json
// @Param request body createAppointmentRequest true "Appointment data"
// @Success 201 {object} appointmentResponse
// @Failure 400 {object} ProblemDetails
// @Failure 409 {object} ProblemDetails
// @Router /appointments [post]
func (handler *AppointmentHandler) Create(writer http.ResponseWriter, request *http.Request) {
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

	var req createAppointmentRequest

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

	if validationErrs := handler.validator.Validate(req); validationErrs != nil {
		// Using the ValidationErrorContext structure that we created earlier
		err := &ValidationErrorContext{
			Err:     domain.ErrInvalidInput,
			Details: validationErrs,
		}
		RespondError(writer, request, handler.logger, err)
		return
	}

	appointment, err := handler.usecase.Create(
		request.Context(),
		req.StoreID,
		req.ServiceID,
		req.ClientID,
		req.StartTime,
		req.DurationMinutes,
	)

	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusCreated, appointmentResponse{
		ID:        appointment.ID,
		StoreID:   appointment.StoreID,
		ServiceID: appointment.ServiceID,
		UserID:    appointment.UserID,
		StartTime: appointment.StartTime,
		EndTime:   appointment.EndTime,
		Status:    appointment.Status,
	}); err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"error trying to write JSON response",
			"error", err,
			"method", request.Method,
			"path", request.URL.Path,
		)
	}
}
