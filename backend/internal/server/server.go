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
    projectsService *services.ProjectsService
	imageService    *services.ImageGenerationService
	storageService  *services.StorageService
	pagesService    *services.PagesService
	agentCopilotMgr *services.AgentAsyncCopilotManager
}

func NewFiberServer(
	db database.Service,
	authService *services.AuthService,
	blogService *services.ArticleService,
    projectsService *services.ProjectsService,
	imageService *services.ImageGenerationService,
	storageService *services.StorageService,
	pagesService *services.PagesService,
	agentCopilotMgr *services.AgentAsyncCopilotManager,
) *FiberServer {
	server := &FiberServer{
		App:             fiber.New(),
		db:              db,
		authService:     authService,
		blogService:     blogService,
        projectsService: projectsService,
		imageService:    imageService,
		storageService:  storageService,
		pagesService:    pagesService,
		agentCopilotMgr: agentCopilotMgr,
	}

	// Register routes
	router.RegisterRoutes(server.App, router.RouteDeps{
		AuthService:     authService,
		BlogService:     blogService,
        ProjectsService: projectsService,
		ImageService:    imageService,
		StorageService:  storageService,
		PagesService:    pagesService,
		AgentCopilotMgr: agentCopilotMgr,
	})

	return server
}
