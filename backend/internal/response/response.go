// Package response provides standardized HTTP response helpers
package response

import (
	"blog-agent-go/backend/internal/errors"
	
	"github.com/gofiber/fiber/v2"
)

// ErrorResponse represents a standardized error response structure.
type ErrorResponse struct {
	Error string                 `json:"error"`
	Code  string                 `json:"code,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// PaginationMeta represents pagination metadata
type PaginationMeta struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// PaginatedResponse represents a paginated response
type PaginatedResponse struct {
	Data       interface{}    `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

// SendError sends an error response with appropriate HTTP status code.
// Handles AppError types with structured error information, defaults to 500 for other errors.
func SendError(c *fiber.Ctx, err error) error {
	if appErr, ok := err.(*errors.AppError); ok {
		return c.Status(appErr.StatusCode).JSON(ErrorResponse{
			Error:   appErr.Message,
			Code:    string(appErr.Code),
			Details: appErr.Details,
		})
	}
	
	// Default to internal server error
	return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
		Error: err.Error(),
		Code:  string(errors.ErrCodeInternal),
	})
}

// SendSuccess sends a successful response with the provided data.
func SendSuccess(c *fiber.Ctx, data interface{}) error {
	return c.JSON(SuccessResponse{
		Data: data,
	})
}

// SendMessage sends a success response with just a message.
func SendMessage(c *fiber.Ctx, message string) error {
	return c.JSON(SuccessResponse{
		Message: message,
	})
}

// SendCreated sends a 201 Created response with the provided data.
func SendCreated(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusCreated).JSON(SuccessResponse{
		Data: data,
	})
}

// SendPaginated sends a paginated response with data and pagination metadata.
func SendPaginated(c *fiber.Ctx, data interface{}, meta PaginationMeta) error {
	return c.JSON(PaginatedResponse{
		Data:       data,
		Pagination: meta,
	})
}

