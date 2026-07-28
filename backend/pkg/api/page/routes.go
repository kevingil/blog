// Package page provides HTTP handlers for page management
package page

import (
	"backend/pkg/api/middleware"

	"github.com/gofiber/fiber/v2"
)

// Register registers page routes on the app
func Register(app *fiber.App) {
	// Public routes
	pages := app.Group("/pages")
	pages.Get("/:slug", GetPageBySlug)

	// Dashboard management (authenticated)
	dashboard := app.Group("/dashboard/pages", middleware.Auth())
	dashboard.Get("/", ListPages)
	dashboard.Get("/:id", GetPageByID)
	dashboard.Post("/", CreatePage)
	dashboard.Put("/:id", UpdatePage)
	dashboard.Delete("/:id", DeletePage)
}
