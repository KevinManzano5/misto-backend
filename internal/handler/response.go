// Package handler contains handlers
package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/KevinManzano5/misto-backend/internal/domain"
	"github.com/KevinManzano5/misto-backend/internal/validator"
)

// errorResponse struct
type errorResponse struct {
	Error   string                      `json:"error"`
	Details []validator.ValidationError `json:"details,omitempty"`
}

// ProblemDetails struct
type ProblemDetails struct {
	Type     string                      `json:"type"`
	Title    string                      `json:"title"`
	Status   int                         `json:"status"`
	Detail   string                      `json:"detail"`
	Instance string                      `json:"instance"`
	Errors   []validator.ValidationError `json:"errors,omitempty"`
}

const problemBaseURL = "https://google.com"

// ValidationErrorContext allows validation details to be conveyed alongside the generic error
type ValidationErrorContext struct {
	Err     error
	Details []validator.ValidationError
}

func (e *ValidationErrorContext) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return "invalid request parameters"
}

func (e *ValidationErrorContext) Unwrap() error {
	return e.Err
}

// RespondError function
func RespondError(writer http.ResponseWriter, request *http.Request, logger *slog.Logger, err error) {
	var problem ProblemDetails
	problem.Instance = request.URL.Path

	switch {
	case errors.Is(err, domain.ErrNotFound):
		problem.Type = problemBaseURL + "/not-found"
		problem.Title = "Resource not found"
		problem.Status = http.StatusNotFound
		problem.Detail = err.Error()
	case errors.Is(err, domain.ErrAlreadyExists):
		problem.Type = problemBaseURL + "/conflict"
		problem.Title = "Resource already exists"
		problem.Status = http.StatusConflict
		problem.Detail = err.Error()
	case errors.Is(err, domain.ErrUnauthorized):
		problem.Type = problemBaseURL + "/unauthorized"
		problem.Title = "Unauthorized"
		problem.Status = http.StatusUnauthorized
		problem.Detail = err.Error()
	case errors.Is(err, domain.ErrForbidden):
		problem.Type = problemBaseURL + "/forbidden"
		problem.Title = "You're not allowed to see this content"
		problem.Status = http.StatusForbidden
		problem.Detail = err.Error()
	case errors.Is(err, domain.ErrInvalidInput):
		problem.Type = problemBaseURL + "/bad-request"
		problem.Title = "Invalid request"
		problem.Status = http.StatusBadRequest
		problem.Detail = err.Error()

		var valCtx *ValidationErrorContext
		if errors.As(err, &valCtx) {
			problem.Errors = valCtx.Details
		}
	default:
		problem.Type = problemBaseURL + "/internal"
		problem.Title = "Internal server error"
		problem.Status = http.StatusInternalServerError
		problem.Detail = "unexpected error"

		logger.ErrorContext(request.Context(), "unclassified error", slog.Any("error", err), slog.String("path", problem.Instance))
	}

	writer.Header().Set("Content-Type", "application/problem+json")
	writer.WriteHeader(problem.Status)

	if encodeErr := json.NewEncoder(writer).Encode(problem); encodeErr != nil {
		logger.Error("error trying to write problem details", slog.String("error", encodeErr.Error()))
	}

}

// RespondValidationError function
func RespondValidationError(writer http.ResponseWriter, errors []validator.ValidationError) error {
	return respondJSON(writer, http.StatusBadRequest, errorResponse{
		Error:   "invalid data",
		Details: errors,
	})
}

// RespondJSON function
func RespondJSON(writer http.ResponseWriter, status int, data any) error {
	return respondJSON(writer, status, data)
}

func respondJSON(writer http.ResponseWriter, status int, data any) error {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)

	return json.NewEncoder(writer).Encode(data)
}
