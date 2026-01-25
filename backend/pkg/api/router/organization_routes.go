package router

import (
	"backend/pkg/api/handler"
	"backend/pkg/api/middleware"

	"github.com/gofiber/fiber/v2"
)

// RegisterOrganizationRoutes registers organization routes
func RegisterOrganizationRoutes(app *fiber.App, deps RouteDeps) {
	// All organization routes require authentication
	orgs := app.Group("/organizations", middleware.AuthMiddleware(deps.AuthService))

	orgs.Get("/", handler.ListOrganizationsHandler(deps.OrganizationService))
	orgs.Post("/", handler.CreateOrganizationHandler(deps.OrganizationService))
	orgs.Get("/:id", handler.GetOrganizationHandler(deps.OrganizationService))
	orgs.Put("/:id", handler.UpdateOrganizationHandler(deps.OrganizationService))
	orgs.Delete("/:id", handler.DeleteOrganizationHandler(deps.OrganizationService))
	orgs.Post("/:id/join", handler.JoinOrganizationHandler(deps.OrganizationService))
	orgs.Post("/leave", handler.LeaveOrganizationHandler(deps.OrganizationService))
}
