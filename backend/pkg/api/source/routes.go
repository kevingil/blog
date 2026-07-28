// Package source provides HTTP handlers for article source management
package source

import (
	"backend/pkg/api/middleware"

	"github.com/gofiber/fiber/v2"
)

// Register registers source routes on the app
func Register(app *fiber.App) {
	// Dashboard sources (authenticated)
	dashboard := app.Group("/dashboard/sources", middleware.Auth())
	dashboard.Get("/", ListAllSources)

	// All source routes require authentication
	sources := app.Group("/sources", middleware.Auth())
	sources.Post("/", CreateSource)
	sources.Post("/scrape", ScrapeAndCreateSource)
	sources.Get("/article/:articleId", GetArticleSources)
	sources.Get("/article/:articleId/search", SearchSimilarSources)
	sources.Get("/:sourceId", GetSource)
	sources.Put("/:sourceId", UpdateSource)
	sources.Delete("/:sourceId", DeleteSource)
}
