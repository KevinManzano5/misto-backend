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

// StoreCategoryHandler struct
type StoreCategoryHandler struct {
	usecase   *usecase.StoreCategoryUseCase
	validator *validator.Validator
	logger    *slog.Logger
}

// NewStoreCategoryHandler constructor
func NewStoreCategoryHandler(usecase *usecase.StoreCategoryUseCase, validator *validator.Validator, logger *slog.Logger) *StoreCategoryHandler {
	return &StoreCategoryHandler{
		usecase:   usecase,
		validator: validator,
		logger:    logger,
	}
}

// storeCategoryResponse struct
type storeCategoryResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ListCategories endpoint (GET /api/v1/store-categories)
//
// @Summary List the store category catalog
// @Description Returns every available store category, for a frontend selection UI.
// @Tags store_categories
// @Produce json
// @Success 200 {array} storeCategoryResponse
// @Router /store-categories [get]
func (handler *StoreCategoryHandler) ListCategories(writer http.ResponseWriter, request *http.Request) {
	categories, err := handler.usecase.ListCategories(request.Context())
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	response := make([]storeCategoryResponse, 0, len(categories))
	for _, category := range categories {
		response = append(response, storeCategoryResponse{
			ID:          category.ID,
			Name:        category.Name,
			Slug:        category.Slug,
			Description: category.Description,
			CreatedAt:   category.CreatedAt,
			UpdatedAt:   category.UpdatedAt,
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

// createStoreCategoryRequest struct
type createStoreCategoryRequest struct {
	Name        string `json:"name" validate:"required,min=2,max=50"`
	Slug        string `json:"slug" validate:"required,slug,max=50"`
	Description string `json:"description" validate:"omitempty,max=1000"`
}

// updateStoreCategoryRequest struct
type updateStoreCategoryRequest struct {
	Name        *string `json:"name" validate:"omitempty,min=2,max=50"`
	Slug        *string `json:"slug" validate:"omitempty,slug,max=50"`
	Description *string `json:"description" validate:"omitempty,max=1000"`
}

// decodeStoreCategoryJSONBody decodes a request body into dest, rejecting unknown fields and
// trailing data.
func decodeStoreCategoryJSONBody(request *http.Request, dest any) error {
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

// Create endpoint (POST /api/v1/store-categories). Adds a new entry to the store category
// catalog. Admin-only.
//
// @Summary Create a store category
// @Description Adds a new entry to the store category catalog.
// @Tags store_categories
// @Accept json
// @Produce json
// @Param request body createStoreCategoryRequest true "Category data"
// @Success 201 {object} storeCategoryResponse
// @Router /store-categories [post]
func (handler *StoreCategoryHandler) Create(writer http.ResponseWriter, request *http.Request) {
	var req createStoreCategoryRequest
	if err := decodeStoreCategoryJSONBody(request, &req); err != nil {
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

	category, err := handler.usecase.Create(request.Context(), req.Name, req.Slug, req.Description)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	response := storeCategoryResponse{
		ID:          category.ID,
		Name:        category.Name,
		Slug:        category.Slug,
		Description: category.Description,
		CreatedAt:   category.CreatedAt,
		UpdatedAt:   category.UpdatedAt,
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

// Update endpoint (PATCH /api/v1/store-categories/{id}). Applies partial changes to a catalog
// entry. Admin-only.
//
// @Summary Update a store category
// @Description Applies partial changes to an existing store category.
// @Tags store_categories
// @Accept json
// @Produce json
// @Param id path string true "Category ID"
// @Param request body updateStoreCategoryRequest true "Fields to update"
// @Success 200 {object} storeCategoryResponse
// @Router /store-categories/{id} [patch]
func (handler *StoreCategoryHandler) Update(writer http.ResponseWriter, request *http.Request) {
	categoryID := request.PathValue("id")

	var req updateStoreCategoryRequest
	if err := decodeStoreCategoryJSONBody(request, &req); err != nil {
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

	category, err := handler.usecase.Update(request.Context(), categoryID, req.Name, req.Slug, req.Description)
	if err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	response := storeCategoryResponse{
		ID:          category.ID,
		Name:        category.Name,
		Slug:        category.Slug,
		Description: category.Description,
		CreatedAt:   category.CreatedAt,
		UpdatedAt:   category.UpdatedAt,
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

// Delete endpoint (DELETE /api/v1/store-categories/{id}). Removes a catalog entry. Admin-only.
// Fails if the category is still assigned to a store or used by a service.
//
// @Summary Delete a store category
// @Description Deletes a store category from the catalog.
// @Tags store_categories
// @Produce json
// @Param id path string true "Category ID"
// @Success 200 {object} map[string]string
// @Router /store-categories/{id} [delete]
func (handler *StoreCategoryHandler) Delete(writer http.ResponseWriter, request *http.Request) {
	categoryID := request.PathValue("id")

	if err := handler.usecase.Delete(request.Context(), categoryID); err != nil {
		RespondError(writer, request, handler.logger, err)
		return
	}

	if err := RespondJSON(writer, http.StatusOK, "category deleted"); err != nil {
		handler.logger.ErrorContext(
			request.Context(),
			"error trying to write JSON response",
			"error", err,
			"method", request.Method,
			"path", request.URL.Path,
		)
	}
}
