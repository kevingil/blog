package router

import (
	"blog-agent-go/backend/internal/controller"
	"blog-agent-go/backend/internal/services"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

type RouteDeps struct {
	AuthService     *services.AuthService
	BlogService     *services.ArticleService
	ImageService    *services.ImageGenerationService
	StorageService  *services.StorageService
	PagesService    *services.PagesService
	AsyncCopilotMgr *services.AsyncCopilotManager
}

func RegisterRoutes(app *fiber.App, deps RouteDeps) {
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS,PATCH",
		AllowHeaders:     "Accept,Authorization,Content-Type",
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Copilot
	app.Post("/agent", controller.WritingCopilotHandler())
	app.Get("/websocket", websocket.New(controller.WebsocketHandler(deps.AsyncCopilotMgr)))

	// Pages
	pages := app.Group("/pages")
	pages.Get(":slug", controller.GetPageBySlugHandler(deps.PagesService))

	// Auth
	auth := app.Group("/auth")
	auth.Post("/login", controller.LoginHandler(deps.AuthService))
	auth.Post("/register", controller.RegisterHandler(deps.AuthService))
	auth.Post("/logout", controller.LogoutHandler())

	protected := auth.Group("", controller.AuthMiddleware(deps.AuthService))
	protected.Put("/account", controller.UpdateAccountHandler(deps.AuthService))
	protected.Put("/password", controller.UpdatePasswordHandler(deps.AuthService))
	protected.Delete("/account", controller.DeleteAccountHandler(deps.AuthService))

	// Blog
	blog := app.Group("/blog")
	blog.Post("/generate", controller.GenerateArticleHandler(deps.BlogService))
	blog.Put(":id/update", controller.UpdateArticleWithContextHandler(deps.BlogService))
	blog.Get("/articles/:slug", controller.GetArticleDataHandler(deps.BlogService))
	blog.Post("/articles/:slug/update", controller.UpdateArticleHandler(deps.BlogService))
	blog.Post("/articles", controller.CreateArticleHandler(deps.BlogService))
	blog.Get("/articles/:id/recommended", controller.GetRecommendedArticlesHandler(deps.BlogService))
	blog.Delete("/articles/:id", controller.DeleteArticleHandler(deps.BlogService))
	blog.Get("/articles", controller.GetArticlesHandler(deps.BlogService))
	blog.Get("/articles/search", controller.SearchArticlesHandler(deps.BlogService))
	blog.Get("/tags/popular", controller.GetPopularTagsHandler(deps.BlogService))

	// Images
	images := app.Group("/images")
	images.Post("/generate", controller.GenerateArticleImageHandler(deps.ImageService))
	images.Get(":requestId", controller.GetImageGenerationHandler(deps.ImageService))
	images.Get(":requestId/status", controller.GetImageGenerationStatusHandler(deps.ImageService))

	// Storage
	storage := app.Group("/storage")
	storage.Get("/files", controller.ListFilesHandler(deps.StorageService))
	storage.Post("/upload", controller.UploadFileHandler(deps.StorageService))
	storage.Delete(":key", controller.DeleteFileHandler(deps.StorageService))
	storage.Post("/folders", controller.CreateFolderHandler(deps.StorageService))
	storage.Put("/folders", controller.UpdateFolderHandler(deps.StorageService))

	// Base
	app.Get("/", controller.HelloWorldHandler())
	app.Get("/health", controller.HealthHandler())
}
