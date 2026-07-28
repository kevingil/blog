// Package worker provides HTTP handlers for worker management
package worker

import (
	"backend/pkg/api/middleware"

	"github.com/gofiber/fiber/v2"
)

// Register registers worker routes on the app
func Register(app *fiber.App) {
	// All worker routes require authentication
	workers := app.Group("/workers", middleware.Auth())

	// Status routes
	workers.Get("/status", GetAllWorkerStatus)
	workers.Get("/running", GetRunningWorkers)

	// Individual worker routes
	workers.Get("/:name/status", GetWorkerStatus)
	workers.Post("/:name/run", RunWorker)
	workers.Post("/:name/stop", StopWorker)
}
