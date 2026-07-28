package validation

import (
	"backend/pkg/api/response"

	"github.com/gofiber/fiber/v2"
)

// ValidateBody is a generic middleware that parses and validates request bodies
// Usage: app.Post("/articles", validation.ValidateBody[dto.CreateArticleRequest](), handler)
func ValidateBody[T any]() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req T

		// Parse body
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, fiber.NewError(fiber.StatusBadRequest, "Invalid request body"))
		}

		// Validate struct
		if err := ValidateStruct(&req); err != nil {
			return response.Error(c, err)
		}

		// Store validated request in locals for handler to access
		c.Locals("validatedBody", req)

		return c.Next()
	}
}

// GetValidatedBody retrieves the validated body from fiber context
// Usage: req := validation.GetValidatedBody[dto.CreateArticleRequest](c)
func GetValidatedBody[T any](c *fiber.Ctx) T {
	return c.Locals("validatedBody").(T)
}

// ValidateQuery is a generic middleware that parses and validates query parameters
func ValidateQuery[T any]() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req T

		// Parse query parameters
		if err := c.QueryParser(&req); err != nil {
			return response.Error(c, fiber.NewError(fiber.StatusBadRequest, "Invalid query parameters"))
		}

		// Validate struct
		if err := ValidateStruct(&req); err != nil {
			return response.Error(c, err)
		}

		// Store validated request in locals for handler to access
		c.Locals("validatedQuery", req)

		return c.Next()
	}
}

// GetValidatedQuery retrieves the validated query from fiber context
func GetValidatedQuery[T any](c *fiber.Ctx) T {
	return c.Locals("validatedQuery").(T)
}
