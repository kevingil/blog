package router

import (
	"backend/pkg/api/handler"
	"backend/pkg/api/middleware"

	"github.com/gofiber/fiber/v2"
)

// RegisterSourceRoutes registers article source routes
func RegisterSourceRoutes(app *fiber.App, deps RouteDeps) {
	// Dashboard sources (authenticated)
	dashboardSources := app.Group("/dashboard/sources", middleware.AuthMiddleware(deps.AuthService))
	dashboardSources.Get("/", handler.ListAllSourcesHandler(deps.SourcesService))

	// All source routes require authentication
	sources := app.Group("/sources", middleware.AuthMiddleware(deps.AuthService))
	sources.Post("/", handler.CreateSourceHandler(deps.SourcesService))
	sources.Post("/scrape", handler.ScrapeAndCreateSourceHandler(deps.SourcesService))
	sources.Get("/article/:articleId", handler.GetArticleSourcesHandler(deps.SourcesService))
	sources.Get("/article/:articleId/search", handler.SearchSimilarSourcesHandler(deps.SourcesService))
	sources.Get("/:sourceId", handler.GetSourceHandler(deps.SourcesService))
	sources.Put("/:sourceId", handler.UpdateSourceHandler(deps.SourcesService))
	sources.Delete("/:sourceId", handler.DeleteSourceHandler(deps.SourcesService))
}
