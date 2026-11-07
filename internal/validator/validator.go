// Package validator contains validators
package validator

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

// Validator struct
type Validator struct {
	validate *validator.Validate
}

// New function
func New() *Validator {
	v := validator.New(validator.WithRequiredStructEnabled())
	_ = v.RegisterValidation("hhmm", validateHHMM)

	return &Validator{validate: v}
}

// validateHHMM checks that a field is a clock time in "HH:MM" or "HH:MM:SS" format.
func validateHHMM(fl validator.FieldLevel) bool {
	value := fl.Field().String()

	if _, err := time.Parse("15:04:05", value); err == nil {
		return true
	}

	_, err := time.Parse("15:04", value)
	return err == nil
}

// ValidationError struct
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Validate method
func (v *Validator) Validate(s any) []ValidationError {
	err := v.validate.Struct(s)

	if err == nil {
		return nil
	}

	validationErrors, ok := err.(validator.ValidationErrors)

	if !ok {
		return []ValidationError{
			{
				Field:   "general",
				Message: err.Error(),
			},
		}
	}

	var errors []ValidationError

	for _, e := range validationErrors {
		errors = append(errors, ValidationError{
			Field:   strings.ToLower(e.Field()),
			Message: translateTag(e),
		})
	}

	return errors
}

func translateTag(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "field mandatory"
	case "email":
		return "must be a valid email"
	case "min":
		return fmt.Sprintf("must have at least %s characters", e.Param())
	case "max":
		return fmt.Sprintf("it must have a maximum of %s characters", e.Param())
	case "hhmm":
		return "must be a valid HH:MM time"
	default:
		return "invalid value"
	}
}
