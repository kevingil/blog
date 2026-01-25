package router

import (
	"backend/pkg/api/handler"
	"backend/pkg/api/middleware"

	"github.com/gofiber/fiber/v2"
)

// RegisterPageRoutes registers page routes
func RegisterPageRoutes(app *fiber.App, deps RouteDeps) {
	// Public routes
	pages := app.Group("/pages")
	pages.Get(":slug", handler.GetPageBySlugHandler(deps.PagesService))

	// Dashboard management (authenticated)
	dashboardPages := app.Group("/dashboard/pages", middleware.AuthMiddleware(deps.AuthService))
	dashboardPages.Get("/", handler.ListPagesHandler(deps.PagesService))
	dashboardPages.Get("/:id", handler.GetPageByIDHandler(deps.PagesService))
	dashboardPages.Post("/", handler.CreatePageHandler(deps.PagesService))
	dashboardPages.Put("/:id", handler.UpdatePageHandler(deps.PagesService))
	dashboardPages.Delete("/:id", handler.DeletePageHandler(deps.PagesService))
}
