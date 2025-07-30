package server

import (
	"blog-agent-go/backend/internal/database"
	"blog-agent-go/backend/internal/router"
	"blog-agent-go/backend/internal/services"

	"github.com/gofiber/fiber/v2"
)

type FiberServer struct {
	App             *fiber.App
	db              database.Service
	authService     *services.AuthService
	blogService     *services.ArticleService
	imageService    *services.ImageGenerationService
	storageService  *services.StorageService
	pagesService    *services.PagesService
	asyncCopilotMgr *services.AsyncCopilotManager
}

func NewFiberServer(
	db database.Service,
	authService *services.AuthService,
	blogService *services.ArticleService,
	imageService *services.ImageGenerationService,
	storageService *services.StorageService,
	pagesService *services.PagesService,
	asyncCopilotMgr *services.AsyncCopilotManager,
) *FiberServer {
	server := &FiberServer{
		App:             fiber.New(),
		db:              db,
		authService:     authService,
		blogService:     blogService,
		imageService:    imageService,
		storageService:  storageService,
		pagesService:    pagesService,
		asyncCopilotMgr: asyncCopilotMgr,
	}

	// Register routes
	router.RegisterRoutes(server.App, router.RouteDeps{
		AuthService:     authService,
		BlogService:     blogService,
		ImageService:    imageService,
		StorageService:  storageService,
		PagesService:    pagesService,
		AsyncCopilotMgr: asyncCopilotMgr,
	})

	return server
}
