package router

import (
	"backend/pkg/api/handler"
	"backend/pkg/api/middleware"

	"github.com/gofiber/fiber/v2"
)

// RegisterProjectRoutes registers project routes
func RegisterProjectRoutes(app *fiber.App, deps RouteDeps) {
	projects := app.Group("/projects")

	// Public routes
	projects.Get("/", handler.ListProjectsHandler(deps.ProjectsService))
	projects.Get(":id", handler.GetProjectHandler(deps.ProjectsService))

	// Protected routes (require authentication)
	projectsProtected := projects.Group("", middleware.AuthMiddleware(deps.AuthService))
	projectsProtected.Post("/", handler.CreateProjectHandler(deps.ProjectsService))
	projectsProtected.Put(":id", handler.UpdateProjectHandler(deps.ProjectsService))
	projectsProtected.Delete(":id", handler.DeleteProjectHandler(deps.ProjectsService))
}
