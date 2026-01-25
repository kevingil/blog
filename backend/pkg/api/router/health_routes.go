package router

import (
	"backend/pkg/api/handler"
	"backend/pkg/database"

	"github.com/gofiber/fiber/v2"
)

// RegisterHealthRoutes registers health check routes
func RegisterHealthRoutes(app *fiber.App, dbService database.Service) {
	health := app.Group("/health")

	// Kubernetes liveness probe
	health.Get("/live", handler.LivenessHandler())

	// Kubernetes readiness probe
	health.Get("/ready", handler.ReadinessHandler(dbService))

	// Full health check (default /health route)
	app.Get("/health", handler.FullHealthHandler(dbService))
}
