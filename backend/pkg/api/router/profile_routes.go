package router

import (
	"backend/pkg/api/handler"
	"backend/pkg/api/middleware"

	"github.com/gofiber/fiber/v2"
)

// RegisterProfileRoutes registers profile and site settings routes
func RegisterProfileRoutes(app *fiber.App, deps RouteDeps) {
	profile := app.Group("/profile")

	// Public routes
	profile.Get("/public", handler.GetPublicProfileHandler(deps.ProfileService))

	// Protected routes
	profileProtected := profile.Group("", middleware.AuthMiddleware(deps.AuthService))
	profileProtected.Get("/", handler.GetMyProfileHandler(deps.ProfileService))
	profileProtected.Put("/", handler.UpdateProfileHandler(deps.ProfileService))
	profileProtected.Get("/settings", handler.GetSiteSettingsHandler(deps.ProfileService))
	profileProtected.Put("/settings", handler.UpdateSiteSettingsHandler(deps.ProfileService))
}
