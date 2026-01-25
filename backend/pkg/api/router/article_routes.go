package router

import (
	"backend/pkg/api/handler"
	"backend/pkg/api/middleware"

	"github.com/gofiber/fiber/v2"
)

// RegisterArticleRoutes registers blog/article routes
func RegisterArticleRoutes(app *fiber.App, deps RouteDeps) {
	blog := app.Group("/blog")

	// Public routes
	blog.Get("/articles/search", handler.SearchArticlesHandler(deps.BlogService))
	blog.Get("/articles/:slug", handler.GetArticleDataHandler(deps.BlogService))
	blog.Get("/articles/:id/recommended", handler.GetRecommendedArticlesHandler(deps.BlogService))
	blog.Get("/articles", handler.GetArticlesHandler(deps.BlogService))
	blog.Get("/tags/popular", handler.GetPopularTagsHandler(deps.BlogService))

	// Protected routes (require authentication)
	blogProtected := blog.Group("", middleware.AuthMiddleware(deps.AuthService))
	blogProtected.Post("/generate", handler.GenerateArticleHandler(deps.BlogService))
	blogProtected.Put(":id/update", handler.UpdateArticleWithContextHandler(deps.BlogService))
	blogProtected.Post("/articles/:slug/update", handler.UpdateArticleHandler(deps.BlogService))
	blogProtected.Post("/articles", handler.CreateArticleHandler(deps.BlogService))
	blogProtected.Delete("/articles/:id", handler.DeleteArticleHandler(deps.BlogService))
}
