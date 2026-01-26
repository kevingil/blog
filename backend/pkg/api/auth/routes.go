// Package auth provides HTTP handlers for authentication
package auth

import (
	"backend/pkg/api/middleware"

	"github.com/gofiber/fiber/v2"
)

// Register registers authentication routes on the app
func Register(app *fiber.App) {
	auth := app.Group("/auth")

	// Public routes
	auth.Post("/login", Login)
	auth.Post("/register", RegisterHandler)
	auth.Post("/logout", Logout)

	// Protected routes (require authentication)
	protected := auth.Group("", middleware.Auth())
	protected.Put("/account", UpdateAccount)
	protected.Put("/password", UpdatePassword)
	protected.Delete("/account", DeleteAccount)
}
