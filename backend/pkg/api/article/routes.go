// Package article provides HTTP handlers for article/blog management
package article

import (
	"backend/pkg/api/middleware"

	"github.com/gofiber/fiber/v2"
)

// Register registers article routes on the app
func Register(app *fiber.App) {
	blog := app.Group("/blog")

	// Public routes
	blog.Get("/articles/search", SearchArticles)
	blog.Get("/articles/:slug", GetArticleData)
	blog.Get("/articles/:id/recommended", GetRecommendedArticles)
	blog.Get("/articles", GetArticles)
	blog.Get("/tags/popular", GetPopularTags)

	// Protected routes (require authentication)
	protected := blog.Group("", middleware.Auth())
	protected.Post("/generate", GenerateArticle)
	protected.Put("/:id/update", UpdateArticleWithContext)
	protected.Post("/articles/:slug/update", UpdateArticle)
	protected.Post("/articles", CreateArticle)
	protected.Delete("/articles/:id", DeleteArticle)

	// Version management routes (protected)
	protected.Post("/articles/:slug/publish", PublishArticle)
	protected.Post("/articles/:slug/unpublish", UnpublishArticle)
	protected.Get("/articles/:slug/versions", ListVersions)
	protected.Get("/articles/versions/:versionId", GetVersion)
	protected.Post("/articles/:slug/revert/:versionId", RevertToVersion)
}
