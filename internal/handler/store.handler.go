package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/KevinManzano5/misto-backend/internal/domain"
	"github.com/KevinManzano5/misto-backend/internal/middlewares"
	"github.com/KevinManzano5/misto-backend/internal/usecase"
	"github.com/KevinManzano5/misto-backend/internal/validator"
)

// StoreHandler struct
type StoreHandler struct {
	usecase         *usecase.StoreUseCase
	categoryUsecase *usecase.StoreCategoryAssignmentUseCase
	validator       *validator.Validator
	logger          *slog.Logger
}

// NewStoreHandler constructor
func NewStoreHandler(usecase *usecase.StoreUseCase, categoryUsecase *usecase.StoreCategoryAssignmentUseCase, validator *validator.Validator, logger *slog.Logger) *StoreHandler {
	return &StoreHandler{
		usecase:         usecase,
		categoryUsecase: categoryUsecase,
		validator:       validator,
		logger:          logger,
	}
}

type createStoreRequest struct {
	Name                string `json:"name" validate:"required,min=2,max=30"`
	Description         string `json:"description" validate:"required"`
	AppointmentDuration int32  `json:"appointment_duration" validate:"omitempty,min=1"`
	// Timezone is the store's IANA timezone (e.g. "America/Bogota"). Defaults to "America/Bogota" when omitted.
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
	// PhoneNumber is the store's contact phone number, in E.164 format (e.g. "+573001234567").
	PhoneNumber string `json:"phone_number" validate:"omitempty,e164"`
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
	PhoneNumber              *string `json:"phone_number" validate:"omitempty,e164"`
}

type storeResponse struct {
	ID                       string                  `json:"id"`
	UserID                   string                  `json:"user_id"`
	Name                     string                  `json:"name"`
	Slug                     string                  `json:"slug"`
	Description              string                  `json:"description"`
	IsActive                 bool                    `json:"is_active"`
	AppointmentDuration      int32                   `json:"appointment_duration"`
	Timezone                 string                  `json:"timezone"`
	AutoConfirmAppointments  bool                    `json:"auto_confirm_appointments"`
	Email                    string                  `json:"email"`
	Address                  string                  `json:"address"`
	AllowSameDayAppointments bool                    `json:"allow_same_day_appointments"`
	PhoneNumber              string                  `json:"phone_number"`
	Categories               []storeCategoryResponse `json:"categories"`
	AverageRating            float64                 `json:"average_rating"`
	RatingCount              int64                   `json:"rating_count"`
	CreatedAt                time.Time               `json:"created_at"`
	UpdatedAt                time.Time               `json:"updated_at"`
}

// replaceStoreCategoriesRequest struct
type replaceStoreCategoriesRequest struct {
	CategoryIDs []string `json:"category_ids" validate:"required,max=5,dive,uuid"`
}

// uuidPattern matches a canonical (8-4-4-4-12 hex) UUID, case-insensitively.
var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// isUUID reports whether value looks like a UUID, as opposed to a store slug.
func isUUID(value string) bool {
	return uuidPattern.MatchString(value)
}

// parseDayOfWeek converts the {day} path value to a valid day-of-week (0=Sunday..6=Saturday).
func parseDayOfWeek(value string) (int32, error) {
	day, err := strconv.Atoi(value)
	if err != nil || day < 0 || day > 6 {
		return 0, domain.ErrInvalidInput
	}

	return int32(day), nil
}

func toStoreCategoryResponse(category *domain.StoreCategory) storeCategoryResponse {
	return storeCategoryResponse{
		ID:          category.ID,
		Name:        category.Name,
		Slug:        category.Slug,
		Description: category.Description,
		CreatedAt:   category.CreatedAt,
		UpdatedAt:   category.UpdatedAt,
	}
}

// buildStoreResponse composes the full store response, embedding its assigned categories and
// rating summary so callers don't need separate requests to see them. ratingSummary may be nil,
// in which case the response's rating fields are left at their zero value.
func (handler *StoreHandler) buildStoreResponse(ctx context.Context, store *domain.Store, ratingSummary *domain.StoreRatingSummary) (storeResponse, error) {
	categories, err := handler.categoryUsecase.ListStoreCategories(ctx, store.ID)
	if err != nil {
		return storeResponse{}, fmt.Errorf("listing store categories: %w", err)
	}

	categoryResponses := make([]storeCategoryResponse, 0, len(categories))
	for _, category := range categories {
		categoryResponses = append(categoryResponses, toStoreCategoryResponse(category))
	}

	var averageRating float64
	var ratingCount int64
	if ratingSummary != nil {
		averageRating = ratingSummary.AverageScore
		ratingCount = ratingSummary.RatingCount
	}

	return storeResponse{
		ID:                       store.ID,
		UserID:                   store.UserID,
		Name:                     store.Name,
		Slug:                     store.Slug,
		Description:              store.Description,
		IsActive:                 store.IsActive,
		AppointmentDuration:      store.AppointmentDuration,
		Timezone:                 store.Timezone,
		AutoConfirmAppointments:  store.AutoConfirmAppointments,
		Email:                    store.Email,
		Address:                  store.Address,
		AllowSameDayAppointments: store.AllowSameDayAppointments,
		PhoneNumber:              store.PhoneNumber,
		Categories:               categoryResponses,
		AverageRating:            averageRating,
		RatingCount:              ratingCount,
		CreatedAt:                store.CreatedAt,
		UpdatedAt:                store.UpdatedAt,
	}, nil
}

// buildStoreResponseWithRating fetches store's rating summary and composes its full response. Used
// by every single-store endpoint (Create, Get, Update); ListStores fetches summaries in bulk
// instead to avoid N+1 queries.
func (handler *StoreHandler) buildStoreResponseWithRating(ctx context.Context, store *domain.Store) (storeResponse, error) {
	summary, err := handler.usecase.GetRatingSummary(ctx, store.ID)
	if err != nil {
		return storeResponse{}, fmt.Errorf("getting store rating summary: %w", err)
	}

	return handler.buildStoreResponse(ctx, store, summary)
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

	store, err := handler.usecase.Create(request.Context(), userID, req.Name, req.Description, true, req.AppointmentDuration, req.Timezone, autoConfirmAppointments, req.Email, req.Address, allowSameDayAppointments, req.PhoneNumber)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	response, err := handler.buildStoreResponseWithRating(request.Context(), store)
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

// GetStoreByID endpoint. The {id} path value may be either a store's UUID or its slug.
func (handler *StoreHandler) GetStoreByID(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")

	var (
		store *domain.Store
		err   error
	)

	if isUUID(id) {
		store, err = handler.usecase.GetByID(request.Context(), id)
	} else {
		store, err = handler.usecase.GetBySlug(request.Context(), id)
	}

	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	response, err := handler.buildStoreResponseWithRating(request.Context(), store)
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

	response, err := handler.buildStoreResponseWithRating(request.Context(), store)
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

	if req.Name == nil && req.Description == nil && req.IsActive == nil && req.AppointmentDuration == nil && req.Timezone == nil && req.AutoConfirmAppointments == nil && req.Email == nil && req.Address == nil && req.AllowSameDayAppointments == nil && req.PhoneNumber == nil {
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

	store, err := handler.usecase.Update(request.Context(), id, req.Name, req.Description, req.IsActive, req.AppointmentDuration, req.Timezone, req.AutoConfirmAppointments, req.Email, req.Address, req.AllowSameDayAppointments, req.PhoneNumber)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	response, err := handler.buildStoreResponseWithRating(request.Context(), store)
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

	storeIDs := make([]string, 0, len(stores))
	for _, store := range stores {
		storeIDs = append(storeIDs, store.ID)
	}

	ratingSummaries, err := handler.usecase.ListRatingSummaries(request.Context(), storeIDs)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	response := make([]storeResponse, 0, len(stores))
	for _, store := range stores {
		storeWithSchedules, err := handler.buildStoreResponse(request.Context(), store, ratingSummaries[store.ID])
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

// ReplaceCategories endpoint (PUT /stores/{id}/categories). Replaces the full set of categories
// assigned to a store; only the store's owner may call this.
func (handler *StoreHandler) ReplaceCategories(writer http.ResponseWriter, request *http.Request) {
	storeID := request.PathValue("id")

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

	var req replaceStoreCategoriesRequest
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

	categories, err := handler.categoryUsecase.ReplaceCategories(request.Context(), storeID, middlewares.GetUserID(request), req.CategoryIDs)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	response := make([]storeCategoryResponse, 0, len(categories))
	for _, category := range categories {
		response = append(response, toStoreCategoryResponse(category))
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
