package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/KevinManzano5/misto-backend/internal/usecase"
)

// StoreCategoryHandler struct
type StoreCategoryHandler struct {
	usecase *usecase.StoreCategoryUseCase
	logger  *slog.Logger
}

// NewStoreCategoryHandler constructor
func NewStoreCategoryHandler(usecase *usecase.StoreCategoryUseCase, logger *slog.Logger) *StoreCategoryHandler {
	return &StoreCategoryHandler{
		usecase: usecase,
		logger:  logger,
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
