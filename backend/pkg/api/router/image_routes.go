package router

import (
	"backend/pkg/api/handler"
	"backend/pkg/api/middleware"

	"github.com/gofiber/fiber/v2"
)

// RegisterImageRoutes registers image generation routes
func RegisterImageRoutes(app *fiber.App, deps RouteDeps) {
	// All image routes require authentication
	images := app.Group("/images", middleware.AuthMiddleware(deps.AuthService))

	images.Post("/generate", handler.GenerateArticleImageHandler(deps.ImageService))
	images.Get(":requestId", handler.GetImageGenerationHandler(deps.ImageService))
	images.Get(":requestId/status", handler.GetImageGenerationStatusHandler(deps.ImageService))
}
