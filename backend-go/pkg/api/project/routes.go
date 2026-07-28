// Package project provides HTTP handlers for project management
package project

import (
	"backend/pkg/api/middleware"

	"github.com/gofiber/fiber/v2"
)

// Register registers project routes on the app
func Register(app *fiber.App) {
	projects := app.Group("/projects")

	// Public routes
	projects.Get("/", ListProjects)
	projects.Get("/:id", GetProject)

	// Protected routes (require authentication)
	protected := projects.Group("", middleware.Auth())
	protected.Post("/", CreateProject)
	protected.Put("/:id", UpdateProject)
	protected.Delete("/:id", DeleteProject)
}
