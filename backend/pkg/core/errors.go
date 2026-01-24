// Package core provides shared domain errors and types
package core

import "errors"

// Domain errors - these have no HTTP status codes
// The API layer is responsible for mapping these to HTTP responses
var (
	ErrNotFound      = errors.New("resource not found")
	ErrAlreadyExists = errors.New("resource already exists")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("access forbidden")
	ErrValidation    = errors.New("validation failed")
	ErrInvalidInput  = errors.New("invalid input")
	ErrInternal      = errors.New("internal error")
)

// ValidationError represents a field-level validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// ValidationErrors is a collection of validation errors
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "validation failed"
	}
	return e[0].Error()
}

// NewValidationError creates a new validation error for a specific field
func NewValidationError(field, message string) ValidationError {
	return ValidationError{Field: field, Message: message}
}
