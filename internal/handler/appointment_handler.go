// Package handler contains handlers
package handler

import (
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
	StoreID   string    `json:"store_id" validate:"required,uuid"`
	StartTime time.Time `json:"start_time" validate:"required"`
}

// appointmentResponse struct
type appointmentResponse struct {
	ID        string    `json:"id"`
	StoreID   string    `json:"store_id"`
	UserID    string    `json:"user_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Status    string    `json:"status"`
}

// Create method
//
// @Summary Create Appointment
// @Description Schedule a new appointment at a store
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
		fmt.Println(err)

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
		middlewares.GetUserID(request),
		req.StartTime,
	)

	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusCreated, appointmentResponse{
		ID:        appointment.ID,
		StoreID:   appointment.StoreID,
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

// GetAvailableSlots endpoint (e.g., GET /stores/{storeId}/availability)
//
// @Summary Get store availability
// @Description List the available start times for a store on a given date, based on its default appointment duration
// @Tags appointments
// @Produce json
// @Param storeId path string true "Store ID"
// @Param date query string true "Date (YYYY-MM-DD)"
// @Success 200 {array} string
// @Failure 400 {object} ProblemDetails
// @Router /stores/{storeId}/availability [get]
func (handler *AppointmentHandler) GetAvailableSlots(writer http.ResponseWriter, request *http.Request) {
	storeID := request.PathValue("storeId")
	dateStr := request.URL.Query().Get("date")

	if dateStr == "" {
		RespondError(writer, request, handler.logger, domain.ErrInvalidInput)
		return
	}

	slots, err := handler.usecase.GetAvailableSlots(request.Context(), storeID, dateStr)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusOK, slots); err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"error trying to write JSON response",
			"error", err,
			"method", request.Method,
			"path", request.URL.Path,
		)
	}
}

// bookedAppointmentResponse struct
type bookedAppointmentResponse struct {
	ID        string    `json:"id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Status    string    `json:"status"`
}

// GetBookedAppointments endpoint (e.g., GET /stores/{storeId}/appointments)
//
// @Summary Get a store's booked appointments
// @Description Lists the appointments already reserved at a store on a given date. Only the store's owner may call this.
// @Tags appointments
// @Produce json
// @Param storeId path string true "Store ID"
// @Param date query string true "Date (YYYY-MM-DD)"
// @Success 200 {array} bookedAppointmentResponse
// @Failure 400 {object} ProblemDetails
// @Failure 403 {object} ProblemDetails
// @Router /stores/{storeId}/appointments [get]
func (handler *AppointmentHandler) GetBookedAppointments(writer http.ResponseWriter, request *http.Request) {
	storeID := request.PathValue("storeId")
	dateStr := request.URL.Query().Get("date")

	if dateStr == "" {
		RespondError(writer, request, handler.logger, domain.ErrInvalidInput)
		return
	}

	appointments, err := handler.usecase.GetBookedAppointments(request.Context(), storeID, middlewares.GetUserID(request), dateStr)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	response := make([]bookedAppointmentResponse, 0, len(appointments))
	for _, appointment := range appointments {
		response = append(response, bookedAppointmentResponse{
			ID:        appointment.ID,
			StartTime: appointment.StartTime,
			EndTime:   appointment.EndTime,
			Status:    appointment.Status,
		})
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
