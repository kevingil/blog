// Package api provides the main router and blueprint registration
package api

import (
	"backend/pkg/api/agent"
	"backend/pkg/api/article"
	"backend/pkg/api/auth"
	"backend/pkg/api/middleware"
	"backend/pkg/api/organization"
	"backend/pkg/api/page"
	"backend/pkg/api/profile"
	"backend/pkg/api/project"
	"backend/pkg/api/source"

	"github.com/gofiber/fiber/v2"
)

// Init initializes all API services (call this before RegisterRoutes)
func Init() {
	article.InitArticleService()
	auth.InitAuthService()
	organization.InitOrganizationService()
	page.InitPagesService()
	source.InitSourcesService()
	profile.InitProfileService()
}

// RegisterRoutes registers all API routes on the app
func RegisterRoutes(app *fiber.App) {
	// Global middleware
	app.Use(middleware.Recovery())
	app.Use(middleware.RequestLogger())
	app.Use(middleware.CORS())
	app.Use(middleware.Security())

	// Register blueprints
	article.Register(app)
	project.Register(app)
	organization.Register(app)
	page.Register(app)
	source.Register(app)
	profile.Register(app)
	auth.Register(app)
	agent.Register(app)

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Base route
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Blog Agent API"})
	})
}
