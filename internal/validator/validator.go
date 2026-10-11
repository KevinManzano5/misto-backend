// Package validator contains validators
package validator

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validator struct
type Validator struct {
	validate *validator.Validate
}

// New function
func New() *Validator {
	return &Validator{
		validate: validator.New(validator.WithRequiredStructEnabled()),
	}
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
	default:
		return "invalid value"
	}
}
