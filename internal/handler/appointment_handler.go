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
	ID              string    `json:"id"`
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	Status          string    `json:"status"`
	ClientUsername  string    `json:"client_username"`
	ClientEmail     string    `json:"client_email"`
	ClientFirstName string    `json:"client_first_name"`
	ClientLastName  string    `json:"client_last_name"`
}

// GetBookedAppointments endpoint (e.g., GET /stores/{storeId}/appointments)
//
// @Summary Get a store's booked appointments
// @Description Lists every appointment ever booked at a store, any status, with the booking client's basic profile. Only the store's owner may call this. Optionally restrict results to appointments starting within [start_date, end_date] (either may be omitted to leave that side open).
// @Tags appointments
// @Produce json
// @Param storeId path string true "Store ID"
// @Param start_date query string false "Only appointments starting on/after this date (YYYY-MM-DD, store's timezone)"
// @Param end_date query string false "Only appointments starting on/before this date (YYYY-MM-DD, store's timezone)"
// @Success 200 {array} bookedAppointmentResponse
// @Failure 400 {object} ProblemDetails
// @Failure 403 {object} ProblemDetails
// @Router /stores/{storeId}/appointments [get]
func (handler *AppointmentHandler) GetBookedAppointments(writer http.ResponseWriter, request *http.Request) {
	storeID := request.PathValue("storeId")
	startDate := request.URL.Query().Get("start_date")
	endDate := request.URL.Query().Get("end_date")

	appointments, err := handler.usecase.GetBookedAppointments(request.Context(), storeID, middlewares.GetUserID(request), startDate, endDate)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	response := make([]bookedAppointmentResponse, 0, len(appointments))
	for _, appointment := range appointments {
		response = append(response, bookedAppointmentResponse{
			ID:              appointment.ID,
			StartTime:       appointment.StartTime,
			EndTime:         appointment.EndTime,
			Status:          appointment.Status,
			ClientUsername:  appointment.ClientUsername,
			ClientEmail:     appointment.ClientEmail,
			ClientFirstName: appointment.ClientFirstName,
			ClientLastName:  appointment.ClientLastName,
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

// updateAppointmentStatusRequest struct
type updateAppointmentStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=confirmed cancelled"`
}

// myAppointmentResponse struct
type myAppointmentResponse struct {
	ID        string    `json:"id"`
	StoreID   string    `json:"store_id"`
	StoreName string    `json:"store_name"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Status    string    `json:"status"`
}

// GetMyAppointments endpoint (e.g., GET /appointments)
//
// @Summary Get the authenticated user's own appointments
// @Description Lists every appointment ever booked by the authenticated user, any status, with the basic info of the store it was booked at.
// @Tags appointments
// @Produce json
// @Success 200 {array} myAppointmentResponse
// @Router /appointments [get]
func (handler *AppointmentHandler) GetMyAppointments(writer http.ResponseWriter, request *http.Request) {
	appointments, err := handler.usecase.ListMyAppointments(request.Context(), middlewares.GetUserID(request))
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	response := make([]myAppointmentResponse, 0, len(appointments))
	for _, appointment := range appointments {
		response = append(response, myAppointmentResponse{
			ID:        appointment.ID,
			StoreID:   appointment.StoreID,
			StoreName: appointment.StoreName,
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

// UpdateStatus endpoint (e.g., PATCH /appointments/{id}/status)
//
// @Summary Approve or reject a pending appointment
// @Description Moves a pending appointment to confirmed (approve) or cancelled (reject). Only the store's owner may call this.
// @Tags appointments
// @Accept json
// @Produce json
// @Param id path string true "Appointment ID"
// @Param request body updateAppointmentStatusRequest true "New status"
// @Success 200 {object} appointmentResponse
// @Failure 400 {object} ProblemDetails
// @Failure 403 {object} ProblemDetails
// @Router /appointments/{id}/status [patch]
func (handler *AppointmentHandler) UpdateStatus(writer http.ResponseWriter, request *http.Request) {
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

	var req updateAppointmentStatusRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		RespondError(writer, request, handler.logger, domain.ErrInvalidInput)
		return
	}

	if validationErrs := handler.validator.Validate(req); validationErrs != nil {
		err := &ValidationErrorContext{
			Err:     domain.ErrInvalidInput,
			Details: validationErrs,
		}
		RespondError(writer, request, handler.logger, err)
		return
	}

	appointment, err := handler.usecase.UpdateStatus(request.Context(), id, middlewares.GetUserID(request), req.Status)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusOK, appointmentResponse{
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

// Cancel endpoint (e.g., PATCH /appointments/{id}/cancel)
//
// @Summary Cancel an appointment
// @Description Moves a pending or confirmed appointment to cancelled. Either the client who booked it or the store's owner may call this.
// @Tags appointments
// @Produce json
// @Param id path string true "Appointment ID"
// @Success 200 {object} appointmentResponse
// @Failure 400 {object} ProblemDetails
// @Failure 403 {object} ProblemDetails
// @Router /appointments/{id}/cancel [patch]
func (handler *AppointmentHandler) Cancel(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")

	appointment, err := handler.usecase.Cancel(request.Context(), id, middlewares.GetUserID(request))
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusOK, appointmentResponse{
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
