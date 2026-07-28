// Package profile provides HTTP handlers for user profile and settings management
package profile

import (
	"backend/pkg/api/middleware"

	"github.com/gofiber/fiber/v2"
)

// Register registers profile routes on the app
func Register(app *fiber.App) {
	profile := app.Group("/profile")

	// Public routes
	profile.Get("/public", GetPublicProfile)

	// Protected routes
	protected := profile.Group("", middleware.Auth())
	protected.Get("/", GetMyProfile)
	protected.Put("/", UpdateProfile)
	protected.Get("/settings", GetSiteSettings)
	protected.Put("/settings", UpdateSiteSettings)
}
