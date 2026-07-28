// Package core provides shared domain errors and types
package core

import (
	"errors"
	"fmt"
)

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
	ErrDatabase      = errors.New("database error")
	ErrExternal      = errors.New("external service error")
	ErrTokenExpired  = errors.New("token expired")
	ErrTokenInvalid  = errors.New("invalid token")
)

// WrappedError wraps a sentinel error with a custom message
type WrappedError struct {
	Err     error
	Message string
}

func (e *WrappedError) Error() string {
	return e.Message
}

func (e *WrappedError) Unwrap() error {
	return e.Err
}

// Helper functions to create errors with custom messages

// NotFoundError creates a not found error for a specific resource
func NotFoundError(resource string) error {
	return &WrappedError{
		Err:     ErrNotFound,
		Message: fmt.Sprintf("%s not found", resource),
	}
}

// AlreadyExistsError creates an already exists error for a specific resource
func AlreadyExistsError(resource string) error {
	return &WrappedError{
		Err:     ErrAlreadyExists,
		Message: fmt.Sprintf("%s already exists", resource),
	}
}

// UnauthorizedError creates an unauthorized error with a custom message
func UnauthorizedError(message string) error {
	if message == "" {
		message = "unauthorized access"
	}
	return &WrappedError{
		Err:     ErrUnauthorized,
		Message: message,
	}
}

// ForbiddenError creates a forbidden error with a custom message
func ForbiddenError(message string) error {
	if message == "" {
		message = "access forbidden"
	}
	return &WrappedError{
		Err:     ErrForbidden,
		Message: message,
	}
}

// InvalidInputError creates an invalid input error with a custom message
func InvalidInputError(message string) error {
	return &WrappedError{
		Err:     ErrInvalidInput,
		Message: message,
	}
}

// InternalError creates an internal error with a custom message
func InternalError(message string) error {
	if message == "" {
		message = "internal server error"
	}
	return &WrappedError{
		Err:     ErrInternal,
		Message: message,
	}
}

// DatabaseError creates a database error with a custom message
func DatabaseError(message string) error {
	return &WrappedError{
		Err:     ErrDatabase,
		Message: message,
	}
}

// ExternalError creates an external service error with a custom message
func ExternalError(message string) error {
	return &WrappedError{
		Err:     ErrExternal,
		Message: message,
	}
}

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
