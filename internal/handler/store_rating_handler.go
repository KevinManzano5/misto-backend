// Package handler contains handlers
package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/KevinManzano5/misto-backend/internal/domain"
	"github.com/KevinManzano5/misto-backend/internal/middlewares"
	"github.com/KevinManzano5/misto-backend/internal/usecase"
	"github.com/KevinManzano5/misto-backend/internal/validator"
)

// StoreRatingHandler struct
type StoreRatingHandler struct {
	usecase   *usecase.StoreRatingUseCase
	validator *validator.Validator
	logger    *slog.Logger
}

// NewStoreRatingHandler constructor
func NewStoreRatingHandler(usecase *usecase.StoreRatingUseCase, validator *validator.Validator, logger *slog.Logger) *StoreRatingHandler {
	return &StoreRatingHandler{
		usecase:   usecase,
		validator: validator,
		logger:    logger,
	}
}

// createStoreRatingRequest struct
type createStoreRatingRequest struct {
	Score   int32  `json:"score" validate:"required,min=1,max=5"`
	Comment string `json:"comment" validate:"max=1000"`
}

// updateStoreRatingRequest struct
type updateStoreRatingRequest struct {
	Score   int32  `json:"score" validate:"required,min=1,max=5"`
	Comment string `json:"comment" validate:"max=1000"`
}

// storeRatingResponse struct
type storeRatingResponse struct {
	ID            string    `json:"id"`
	AppointmentID string    `json:"appointment_id"`
	StoreID       string    `json:"store_id"`
	UserID        string    `json:"user_id"`
	Score         int32     `json:"score"`
	Comment       string    `json:"comment,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// storeRatingWithClientResponse struct
type storeRatingWithClientResponse struct {
	ID              string    `json:"id"`
	Score           int32     `json:"score"`
	Comment         string    `json:"comment,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	ClientUsername  string    `json:"client_username"`
	ClientFirstName string    `json:"client_first_name"`
	ClientLastName  string    `json:"client_last_name"`
}

// listStoreRatingsResponse struct
type listStoreRatingsResponse struct {
	Ratings    []storeRatingWithClientResponse `json:"ratings"`
	TotalCount int64                           `json:"total_count"`
	Page       int32                           `json:"page"`
	PageSize   int32                           `json:"page_size"`
}

// storeRatingSummaryResponse struct
type storeRatingSummaryResponse struct {
	StoreID      string  `json:"store_id"`
	AverageScore float64 `json:"average_score"`
	RatingCount  int64   `json:"rating_count"`
}

// Create endpoint (e.g., POST /appointments/{appointmentId}/ratings)
//
// @Summary Rate a completed appointment
// @Description Leave a score (1-5) and optional comment for a completed appointment. Only the client who booked it may call this, once.
// @Tags store_ratings
// @Accept json
// @Produce json
// @Param appointmentId path string true "Appointment ID"
// @Param request body createStoreRatingRequest true "Rating data"
// @Success 201 {object} storeRatingResponse
// @Failure 400 {object} ProblemDetails
// @Failure 403 {object} ProblemDetails
// @Failure 409 {object} ProblemDetails
// @Router /appointments/{appointmentId}/ratings [post]
func (handler *StoreRatingHandler) Create(writer http.ResponseWriter, request *http.Request) {
	appointmentID := request.PathValue("appointmentId")

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

	var req createStoreRatingRequest
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
		err := &ValidationErrorContext{
			Err:     domain.ErrInvalidInput,
			Details: validationErrs,
		}
		RespondError(writer, request, handler.logger, err)
		return
	}

	rating, err := handler.usecase.Create(request.Context(), appointmentID, middlewares.GetUserID(request), req.Score, req.Comment)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusCreated, storeRatingResponse{
		ID:            rating.ID,
		AppointmentID: rating.AppointmentID,
		StoreID:       rating.StoreID,
		UserID:        rating.UserID,
		Score:         rating.Score,
		Comment:       rating.Comment,
		CreatedAt:     rating.CreatedAt,
		UpdatedAt:     rating.UpdatedAt,
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

// paginationParams reads optional "page"/"page_size" query parameters, falling back to
// defaultPage/defaultPageSize on absence or a non-numeric value, keeping these public listing
// endpoints permissive rather than rejecting the request.
func paginationParams(request *http.Request, defaultPage, defaultPageSize int32) (page, pageSize int32) {
	page, pageSize = defaultPage, defaultPageSize

	if raw := request.URL.Query().Get("page"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			page = int32(parsed)
		}
	}

	if raw := request.URL.Query().Get("page_size"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			pageSize = int32(parsed)
		}
	}

	return page, pageSize
}

// ListByStore endpoint (e.g., GET /stores/{storeId}/ratings)
//
// @Summary List a store's ratings
// @Description Paginated list of ratings left for a store, newest first, with each client's basic profile.
// @Tags store_ratings
// @Produce json
// @Param storeId path string true "Store ID"
// @Param page query int false "Page number (default 1)"
// @Param page_size query int false "Page size (default 20, max 100)"
// @Success 200 {object} listStoreRatingsResponse
// @Router /stores/{storeId}/ratings [get]
func (handler *StoreRatingHandler) ListByStore(writer http.ResponseWriter, request *http.Request) {
	storeID := request.PathValue("storeId")
	page, pageSize := paginationParams(request, 1, 20)

	ratings, total, err := handler.usecase.ListByStore(request.Context(), storeID, page, pageSize)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	response := listStoreRatingsResponse{
		Ratings:    make([]storeRatingWithClientResponse, 0, len(ratings)),
		TotalCount: total,
		Page:       page,
		PageSize:   pageSize,
	}
	for _, rating := range ratings {
		response.Ratings = append(response.Ratings, storeRatingWithClientResponse{
			ID:              rating.ID,
			Score:           rating.Score,
			Comment:         rating.Comment,
			CreatedAt:       rating.CreatedAt,
			ClientUsername:  rating.ClientUsername,
			ClientFirstName: rating.ClientFirstName,
			ClientLastName:  rating.ClientLastName,
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

// GetSummary endpoint (e.g., GET /stores/{storeId}/ratings/summary)
//
// @Summary Get a store's rating summary
// @Description Average score and total rating count for a store, computed on demand.
// @Tags store_ratings
// @Produce json
// @Param storeId path string true "Store ID"
// @Success 200 {object} storeRatingSummaryResponse
// @Router /stores/{storeId}/ratings/summary [get]
func (handler *StoreRatingHandler) GetSummary(writer http.ResponseWriter, request *http.Request) {
	storeID := request.PathValue("storeId")

	summary, err := handler.usecase.GetSummaryByStore(request.Context(), storeID)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusOK, storeRatingSummaryResponse{
		StoreID:      summary.StoreID,
		AverageScore: summary.AverageScore,
		RatingCount:  summary.RatingCount,
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

// Update endpoint (e.g., PATCH /ratings/{id})
//
// @Summary Edit your own store rating
// @Description Updates the score/comment of a rating you authored.
// @Tags store_ratings
// @Accept json
// @Produce json
// @Param id path string true "Rating ID"
// @Param request body updateStoreRatingRequest true "Updated rating data"
// @Success 200 {object} storeRatingResponse
// @Failure 400 {object} ProblemDetails
// @Failure 403 {object} ProblemDetails
// @Router /ratings/{id} [patch]
func (handler *StoreRatingHandler) Update(writer http.ResponseWriter, request *http.Request) {
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

	var req updateStoreRatingRequest
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

	rating, err := handler.usecase.Update(request.Context(), id, middlewares.GetUserID(request), req.Score, req.Comment)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusOK, storeRatingResponse{
		ID:            rating.ID,
		AppointmentID: rating.AppointmentID,
		StoreID:       rating.StoreID,
		UserID:        rating.UserID,
		Score:         rating.Score,
		Comment:       rating.Comment,
		CreatedAt:     rating.CreatedAt,
		UpdatedAt:     rating.UpdatedAt,
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

// Delete endpoint (e.g., DELETE /ratings/{id})
//
// @Summary Delete your own store rating
// @Tags store_ratings
// @Param id path string true "Rating ID"
// @Success 204
// @Failure 403 {object} ProblemDetails
// @Router /ratings/{id} [delete]
func (handler *StoreRatingHandler) Delete(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")

	if err := handler.usecase.Delete(request.Context(), id, middlewares.GetUserID(request)); err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	writer.WriteHeader(http.StatusNoContent)
}
