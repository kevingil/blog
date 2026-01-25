package handler

import "github.com/gofiber/fiber/v2"

// HelloWorldHandler returns a simple hello world message
// @Summary		Hello World
// @Description	Returns a simple greeting message
// @Tags		base
// @Accept		json
// @Produce		json
// @Success		200	{object}	map[string]string
// @Router		/ [get]
func HelloWorldHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		resp := fiber.Map{"message": "Hello World"}
		return c.JSON(resp)
	}
}

// HealthHandler returns the health status
// @Summary		Health check
// @Description	Returns the health status of the API
// @Tags		base
// @Accept		json
// @Produce		json
// @Success		200	{object}	map[string]string
// @Router		/health [get]
func HealthHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	}
}
