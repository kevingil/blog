package router

import (
	"backend/pkg/api/handler"
	"backend/pkg/api/middleware"

	"github.com/gofiber/fiber/v2"
)

// RegisterAuthRoutes registers authentication routes
func RegisterAuthRoutes(app *fiber.App, deps RouteDeps) {
	auth := app.Group("/auth")

	// Public routes
	auth.Post("/login", handler.LoginHandler(deps.AuthService))
	auth.Post("/register", handler.RegisterHandler(deps.AuthService))
	auth.Post("/logout", handler.LogoutHandler())

	// Protected routes (require authentication)
	protected := auth.Group("", middleware.AuthMiddleware(deps.AuthService))
	protected.Put("/account", handler.UpdateAccountHandler(deps.AuthService))
	protected.Put("/password", handler.UpdatePasswordHandler(deps.AuthService))
	protected.Delete("/account", handler.DeleteAccountHandler(deps.AuthService))
}
