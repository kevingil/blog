// Package organization provides HTTP handlers for organization management
package organization

import (
	"backend/pkg/api/middleware"

	"github.com/gofiber/fiber/v2"
)

// Register registers organization routes on the app
func Register(app *fiber.App) {
	// All organization routes require authentication
	orgs := app.Group("/organizations", middleware.Auth())

	orgs.Get("/", ListOrganizations)
	orgs.Post("/", CreateOrganization)
	orgs.Get("/:id", GetOrganization)
	orgs.Put("/:id", UpdateOrganization)
	orgs.Delete("/:id", DeleteOrganization)
	orgs.Post("/:id/join", JoinOrganization)
	orgs.Post("/leave", LeaveOrganization)
}
