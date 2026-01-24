package response

import (
	"errors"

	"blog-agent-go/backend/internal/core"
	apperrors "blog-agent-go/backend/internal/errors"

	"github.com/gofiber/fiber/v2"
)

// MapCoreError maps core domain errors to HTTP responses
// This provides a clean separation between domain errors and HTTP concerns
func MapCoreError(c *fiber.Ctx, err error) error {
	// First check for core domain errors
	switch {
	case errors.Is(err, core.ErrNotFound):
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Error: err.Error(),
			Code:  "NOT_FOUND",
		})

	case errors.Is(err, core.ErrAlreadyExists):
		return c.Status(fiber.StatusConflict).JSON(ErrorResponse{
			Error: err.Error(),
			Code:  "ALREADY_EXISTS",
		})

	case errors.Is(err, core.ErrUnauthorized):
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
			Error: err.Error(),
			Code:  "UNAUTHORIZED",
		})

	case errors.Is(err, core.ErrForbidden):
		return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
			Error: err.Error(),
			Code:  "FORBIDDEN",
		})

	case errors.Is(err, core.ErrValidation):
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: err.Error(),
			Code:  "VALIDATION_ERROR",
		})

	case errors.Is(err, core.ErrInvalidInput):
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: err.Error(),
			Code:  "INVALID_INPUT",
		})

	case errors.Is(err, core.ErrInternal):
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: "Internal server error",
			Code:  "INTERNAL_ERROR",
		})
	}

	// Check for core.ValidationError
	var validationErr core.ValidationError
	if errors.As(err, &validationErr) {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: validationErr.Error(),
			Code:  "VALIDATION_ERROR",
			Details: map[string]interface{}{
				"field":   validationErr.Field,
				"message": validationErr.Message,
			},
		})
	}

	// Check for core.ValidationErrors (multiple)
	var validationErrs core.ValidationErrors
	if errors.As(err, &validationErrs) {
		details := make(map[string]interface{})
		for _, ve := range validationErrs {
			details[ve.Field] = ve.Message
		}
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   validationErrs.Error(),
			Code:    "VALIDATION_ERROR",
			Details: details,
		})
	}

	// Fall back to legacy AppError handling for backwards compatibility
	if appErr, ok := err.(*apperrors.AppError); ok {
		return c.Status(appErr.StatusCode).JSON(ErrorResponse{
			Error:   appErr.Message,
			Code:    string(appErr.Code),
			Details: appErr.Details,
		})
	}

	// Default to internal server error
	return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
		Error: err.Error(),
		Code:  "INTERNAL_ERROR",
	})
}
