// Package router provides HTTP routing configuration
package router

import (
	"backend/pkg/api/handler"
	"backend/pkg/api/middleware"
	"backend/pkg/api/services"
	"backend/pkg/core/chat"
	"backend/pkg/database"

	"github.com/gofiber/fiber/v2"
)

// RouteDeps holds all service dependencies for routes
type RouteDeps struct {
	DBService           database.Service
	AuthService         *services.AuthService
	BlogService         *services.ArticleService
	ProjectsService     *services.ProjectsService
	ImageService        *services.ImageGenerationService
	StorageService      *services.StorageService
	PagesService        *services.PagesService
	SourcesService      *services.ArticleSourceService
	ChatService         *chat.MessageService
	AgentCopilotMgr     *services.AgentAsyncCopilotManager
	ProfileService      *services.ProfileService
	OrganizationService *services.OrganizationService
}

// RegisterRoutes registers all API routes
func RegisterRoutes(app *fiber.App, deps RouteDeps) {
	// Apply global middleware
	app.Use(middleware.Recovery())
	app.Use(middleware.RequestLogger())
	app.Use(middleware.CORS())
	app.Use(middleware.Security())

	// Register domain-specific routes
	RegisterAuthRoutes(app, deps)
	RegisterArticleRoutes(app, deps)
	RegisterPageRoutes(app, deps)
	RegisterProjectRoutes(app, deps)
	RegisterOrganizationRoutes(app, deps)
	RegisterImageRoutes(app, deps)
	RegisterSourceRoutes(app, deps)
	RegisterProfileRoutes(app, deps)
	RegisterStorageRoutes(app, deps)
	RegisterAgentRoutes(app, deps)

	// Health check routes (with database dependency)
	RegisterHealthRoutes(app, deps.DBService)

	// Base routes
	app.Get("/", handler.HelloWorldHandler())
}
