package router

import (
	"blog-agent-go/backend/internal/controller"
	"blog-agent-go/backend/internal/core/chat"
	"blog-agent-go/backend/internal/middleware"
	"blog-agent-go/backend/internal/services"
	"fmt"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

type RouteDeps struct {
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

func RegisterRoutes(app *fiber.App, deps RouteDeps) {
	// Apply global middleware
	app.Use(middleware.Recovery())
	app.Use(middleware.RequestLogger())
	app.Use(middleware.CORS())
	app.Use(middleware.Security())

	// Copilot - Agent-powered writing assistant (requires authentication)
	app.Post("/agent", middleware.AuthMiddleware(deps.AuthService), controller.AgentCopilotHandler())
	app.Get("/websocket", websocket.New(controller.WebsocketHandler()))

	// Agent - Conversation and artifact management (requires authentication)
	agentGroup := app.Group("/agent", middleware.AuthMiddleware(deps.AuthService))
	agentGroup.Get("/conversations/:articleId", controller.GetConversationHistoryHandler(deps.ChatService))
	
	// Artifact endpoints with chat service injection
	agentGroup.Get("/artifacts/:articleId/pending", func(c *fiber.Ctx) error {
		c.Locals("chatService", deps.ChatService)
		return controller.GetPendingArtifacts()(c)
	})
	agentGroup.Post("/artifacts/:messageId/accept", func(c *fiber.Ctx) error {
		c.Locals("chatService", deps.ChatService)
		return controller.AcceptArtifact()(c)
	})
	agentGroup.Post("/artifacts/:messageId/reject", func(c *fiber.Ctx) error {
		c.Locals("chatService", deps.ChatService)
		return controller.RejectArtifact()(c)
	})

	// Pages - Public routes
	pages := app.Group("/pages")
	pages.Get(":slug", controller.GetPageBySlugHandler(deps.PagesService))
	fmt.Println("✓ Registered public pages routes: GET /pages/:slug")

	// Pages - Dashboard management (authenticated)
	dashboardPages := app.Group("/dashboard/pages", middleware.AuthMiddleware(deps.AuthService))
	dashboardPages.Get("/", controller.ListPagesHandler(deps.PagesService))
	dashboardPages.Get("/:id", controller.GetPageByIDHandler(deps.PagesService))
	dashboardPages.Post("/", controller.CreatePageHandler(deps.PagesService))
	dashboardPages.Put("/:id", controller.UpdatePageHandler(deps.PagesService))
	dashboardPages.Delete("/:id", controller.DeletePageHandler(deps.PagesService))
	fmt.Println("✓ Registered dashboard pages routes:")
	fmt.Println("  - GET    /dashboard/pages")
	fmt.Println("  - GET    /dashboard/pages/:id")
	fmt.Println("  - POST   /dashboard/pages")
	fmt.Println("  - PUT    /dashboard/pages/:id")
	fmt.Println("  - DELETE /dashboard/pages/:id")

	// Auth
	auth := app.Group("/auth")
	auth.Post("/login", controller.LoginHandler(deps.AuthService))
	auth.Post("/register", controller.RegisterHandler(deps.AuthService))
	auth.Post("/logout", controller.LogoutHandler())

	protected := auth.Group("", middleware.AuthMiddleware(deps.AuthService))
	protected.Put("/account", controller.UpdateAccountHandler(deps.AuthService))
	protected.Put("/password", controller.UpdatePasswordHandler(deps.AuthService))
	protected.Delete("/account", controller.DeleteAccountHandler(deps.AuthService))

	// Blog - Public routes
	blog := app.Group("/blog")
	blog.Get("/articles/search", controller.SearchArticlesHandler(deps.BlogService))
	blog.Get("/articles/:slug", controller.GetArticleDataHandler(deps.BlogService))
	blog.Get("/articles/:id/recommended", controller.GetRecommendedArticlesHandler(deps.BlogService))
	blog.Get("/articles", controller.GetArticlesHandler(deps.BlogService))
	blog.Get("/tags/popular", controller.GetPopularTagsHandler(deps.BlogService))

	// Blog - Protected routes (require authentication)
	blogProtected := blog.Group("", middleware.AuthMiddleware(deps.AuthService))
	blogProtected.Post("/generate", controller.GenerateArticleHandler(deps.BlogService))
	blogProtected.Put(":id/update", controller.UpdateArticleWithContextHandler(deps.BlogService))
	blogProtected.Post("/articles/:slug/update", controller.UpdateArticleHandler(deps.BlogService))
	blogProtected.Post("/articles", controller.CreateArticleHandler(deps.BlogService))
	blogProtected.Delete("/articles/:id", controller.DeleteArticleHandler(deps.BlogService))

	// Images (all require authentication)
	images := app.Group("/images", middleware.AuthMiddleware(deps.AuthService))
	images.Post("/generate", controller.GenerateArticleImageHandler(deps.ImageService))
	images.Get(":requestId", controller.GetImageGenerationHandler(deps.ImageService))
	images.Get(":requestId/status", controller.GetImageGenerationStatusHandler(deps.ImageService))

	// Sources - Dashboard management (authenticated)
	dashboardSources := app.Group("/dashboard/sources", middleware.AuthMiddleware(deps.AuthService))
	dashboardSources.Get("/", controller.ListAllSourcesHandler(deps.SourcesService))
	fmt.Println("✓ Registered dashboard sources routes:")
	fmt.Println("  - GET    /dashboard/sources")

	// Sources (all require authentication)
	sources := app.Group("/sources", middleware.AuthMiddleware(deps.AuthService))
	sources.Post("/", controller.CreateSourceHandler(deps.SourcesService))
	sources.Post("/scrape", controller.ScrapeAndCreateSourceHandler(deps.SourcesService))
	sources.Get("/article/:articleId", controller.GetArticleSourcesHandler(deps.SourcesService))
	sources.Get("/article/:articleId/search", controller.SearchSimilarSourcesHandler(deps.SourcesService))
	sources.Get("/:sourceId", controller.GetSourceHandler(deps.SourcesService))
	sources.Put("/:sourceId", controller.UpdateSourceHandler(deps.SourcesService))
	sources.Delete("/:sourceId", controller.DeleteSourceHandler(deps.SourcesService))

	// Projects - Public routes
	projects := app.Group("/projects")
	projects.Get("/", controller.ListProjectsHandler(deps.ProjectsService))
	projects.Get(":id", controller.GetProjectHandler(deps.ProjectsService))

	// Projects - Protected routes (require authentication)
	projectsProtected := projects.Group("", middleware.AuthMiddleware(deps.AuthService))
	projectsProtected.Post("/", controller.CreateProjectHandler(deps.ProjectsService))
	projectsProtected.Put(":id", controller.UpdateProjectHandler(deps.ProjectsService))
	projectsProtected.Delete(":id", controller.DeleteProjectHandler(deps.ProjectsService))

	// Profile - Public routes
	profile := app.Group("/profile")
	profile.Get("/public", controller.GetPublicProfileHandler(deps.ProfileService))
	fmt.Println("✓ Registered public profile routes: GET /profile/public")

	// Profile - Protected routes
	profileProtected := profile.Group("", middleware.AuthMiddleware(deps.AuthService))
	profileProtected.Get("/", controller.GetMyProfileHandler(deps.ProfileService))
	profileProtected.Put("/", controller.UpdateProfileHandler(deps.ProfileService))
	profileProtected.Get("/settings", controller.GetSiteSettingsHandler(deps.ProfileService))
	profileProtected.Put("/settings", controller.UpdateSiteSettingsHandler(deps.ProfileService))
	fmt.Println("✓ Registered protected profile routes:")
	fmt.Println("  - GET    /profile")
	fmt.Println("  - PUT    /profile")
	fmt.Println("  - GET    /profile/settings")
	fmt.Println("  - PUT    /profile/settings")

	// Organizations - Protected routes
	orgs := app.Group("/organizations", middleware.AuthMiddleware(deps.AuthService))
	orgs.Get("/", controller.ListOrganizationsHandler(deps.OrganizationService))
	orgs.Post("/", controller.CreateOrganizationHandler(deps.OrganizationService))
	orgs.Get("/:id", controller.GetOrganizationHandler(deps.OrganizationService))
	orgs.Put("/:id", controller.UpdateOrganizationHandler(deps.OrganizationService))
	orgs.Delete("/:id", controller.DeleteOrganizationHandler(deps.OrganizationService))
	orgs.Post("/:id/join", controller.JoinOrganizationHandler(deps.OrganizationService))
	orgs.Post("/leave", controller.LeaveOrganizationHandler(deps.OrganizationService))
	fmt.Println("✓ Registered organization routes:")
	fmt.Println("  - GET    /organizations")
	fmt.Println("  - POST   /organizations")
	fmt.Println("  - GET    /organizations/:id")
	fmt.Println("  - PUT    /organizations/:id")
	fmt.Println("  - DELETE /organizations/:id")
	fmt.Println("  - POST   /organizations/:id/join")
	fmt.Println("  - POST   /organizations/leave")

	// Storage (all require authentication)
	storage := app.Group("/storage", middleware.AuthMiddleware(deps.AuthService))
	storage.Get("/files", controller.ListFilesHandler(deps.StorageService))
	storage.Post("/upload", controller.UploadFileHandler(deps.StorageService))
	storage.Delete(":key", controller.DeleteFileHandler(deps.StorageService))
	storage.Post("/folders", controller.CreateFolderHandler(deps.StorageService))
	storage.Put("/folders", controller.UpdateFolderHandler(deps.StorageService))

	// Base
	app.Get("/", controller.HelloWorldHandler())
	app.Get("/health", controller.HealthHandler())
}
