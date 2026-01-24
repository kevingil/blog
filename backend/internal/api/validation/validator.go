// Package validation provides request validation using struct tags
package validation

import (
	"blog-agent-go/backend/internal/errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// ValidateStruct validates a struct using struct tags.
// Returns an AppError with validation details if validation fails.
func ValidateStruct(s interface{}) error {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	// Format validation errors
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return errors.NewValidationError("Invalid input")
	}

	// Build error details map
	details := make(map[string]interface{})
	var errorMessages []string
	for _, fieldErr := range validationErrors {
		errorMsg := formatValidationError(fieldErr)
		errorMessages = append(errorMessages, errorMsg)
		details[fieldErr.Field()] = map[string]string{
			"tag":     fieldErr.Tag(),
			"message": errorMsg,
		}
	}

	// Create AppError with detailed field information
	validationErr := errors.NewValidationError(strings.Join(errorMessages, "; "))
	validationErr.WithDetails("fields", details)

	return validationErr
}

// formatValidationError formats a validation error into a human-readable message
func formatValidationError(err validator.FieldError) string {
	field := err.Field()
	tag := err.Tag()

	switch tag {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", field, err.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", field, err.Param())
	case "url":
		return fmt.Sprintf("%s must be a valid URL", field)
	default:
		return fmt.Sprintf("%s failed validation on '%s'", field, tag)
	}
}
