package router

import (
	"backend/pkg/api/handler"

	"github.com/gofiber/fiber/v2"
)

// RegisterBaseRoutes sets up base application routes
func RegisterBaseRoutes(app *fiber.App) {
	app.Get("/", handler.HelloWorldHandler())
	app.Get("/health", handler.HealthHandler())
}
