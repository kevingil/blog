package controller

import "github.com/gofiber/fiber/v2"

func HelloWorldHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		resp := fiber.Map{"message": "Hello World"}
		return c.JSON(resp)
	}
}

func HealthHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	}
}
