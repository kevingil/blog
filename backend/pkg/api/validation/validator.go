// Package validation provides request validation using struct tags
package validation

import (
	"backend/pkg/core"
	"fmt"
	"regexp"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

// slugRegex validates URL-friendly slugs (lowercase letters, numbers, hyphens)
var slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func init() {
	validate = validator.New()

	// Register custom validators
	_ = validate.RegisterValidation("slug", validateSlug)
}

// validateSlug validates that a string is a valid URL slug
func validateSlug(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true // Empty is handled by required tag
	}
	return slugRegex.MatchString(value)
}

// ValidateStruct validates a struct using struct tags.
// Returns core.ValidationErrors if validation fails.
func ValidateStruct(s interface{}) error {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	// Format validation errors
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return core.InvalidInputError("Invalid input")
	}

	// Build core ValidationErrors
	var coreErrors core.ValidationErrors
	for _, fieldErr := range validationErrors {
		coreErrors = append(coreErrors, core.ValidationError{
			Field:   fieldErr.Field(),
			Message: formatValidationError(fieldErr),
		})
	}

	return coreErrors
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
	case "slug":
		return fmt.Sprintf("%s must be a valid URL slug (lowercase letters, numbers, and hyphens)", field)
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, err.Param())
	default:
		return fmt.Sprintf("%s failed validation on '%s'", field, tag)
	}
}
