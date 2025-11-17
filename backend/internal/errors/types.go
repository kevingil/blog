// Package errors provides structured error types and error handling for the application
package errors

import "fmt"

// ErrorCode represents a unique error code for identifying error types.
type ErrorCode string

const (
	// Authentication and authorization errors
	ErrCodeUnauthorized     ErrorCode = "UNAUTHORIZED"
	ErrCodeInvalidToken     ErrorCode = "INVALID_TOKEN"
	ErrCodeTokenExpired     ErrorCode = "TOKEN_EXPIRED"
	ErrCodeInvalidAuth      ErrorCode = "INVALID_AUTH"
	
	// Resource errors
	ErrCodeNotFound         ErrorCode = "NOT_FOUND"
	ErrCodeAlreadyExists    ErrorCode = "ALREADY_EXISTS"
	
	// Validation errors
	ErrCodeValidation       ErrorCode = "VALIDATION_ERROR"
	ErrCodeInvalidInput     ErrorCode = "INVALID_INPUT"
	ErrCodeMissingField     ErrorCode = "MISSING_FIELD"
	
	// Server errors
	ErrCodeInternal         ErrorCode = "INTERNAL_ERROR"
	ErrCodeDatabaseError    ErrorCode = "DATABASE_ERROR"
	ErrCodeExternalService  ErrorCode = "EXTERNAL_SERVICE_ERROR"
)

// AppError represents a structured application error with code, message, and HTTP status.
type AppError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	StatusCode int                    `json:"-"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// NewAppError creates a new application error with the specified code, message, and HTTP status.
func NewAppError(code ErrorCode, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Details:    make(map[string]interface{}),
	}
}

// WithDetails adds additional contextual details to the error.
// Returns the error itself for method chaining.
func (e *AppError) WithDetails(key string, value interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// NewUnauthorizedError creates a 401 Unauthorized error.
// If message is empty, uses a default message.
func NewUnauthorizedError(message string) *AppError {
	if message == "" {
		message = "Unauthorized access"
	}
	return NewAppError(ErrCodeUnauthorized, message, 401)
}

// NewNotFoundError creates a 404 Not Found error for the specified resource.
func NewNotFoundError(resource string) *AppError {
	return NewAppError(ErrCodeNotFound, fmt.Sprintf("%s not found", resource), 404)
}

// NewValidationError creates a 400 Bad Request error for validation failures.
func NewValidationError(message string) *AppError {
	return NewAppError(ErrCodeValidation, message, 400)
}

// NewInternalError creates a 500 Internal Server Error.
// If message is empty, uses a default message.
func NewInternalError(message string) *AppError {
	if message == "" {
		message = "Internal server error"
	}
	return NewAppError(ErrCodeInternal, message, 500)
}

// NewAlreadyExistsError creates a 409 Conflict error for the specified resource.
func NewAlreadyExistsError(resource string) *AppError {
	return NewAppError(ErrCodeAlreadyExists, fmt.Sprintf("%s already exists", resource), 409)
}

// NewInvalidInputError creates a 400 Bad Request error for invalid input.
func NewInvalidInputError(message string) *AppError {
	return NewAppError(ErrCodeInvalidInput, message, 400)
}

