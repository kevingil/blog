package server

import (
	"backend/pkg/api/router"
	"backend/pkg/api/services"
	"backend/pkg/core/chat"
	"backend/pkg/database"

	"github.com/gofiber/fiber/v2"
)

// FiberServer represents the Fiber web server
type FiberServer struct {
	App *fiber.App
}

// NewFiberServer creates and configures a new Fiber server with all routes and middleware
func NewFiberServer(
	db database.Service,
	authService *services.AuthService,
	blogService *services.ArticleService,
	projectsService *services.ProjectsService,
	imageService *services.ImageGenerationService,
	storageService *services.StorageService,
	pagesService *services.PagesService,
	sourcesService *services.ArticleSourceService,
	chatService *chat.MessageService,
	agentCopilotMgr *services.AgentAsyncCopilotManager,
	profileService *services.ProfileService,
	organizationService *services.OrganizationService,
) *FiberServer {
	server := &FiberServer{
		App: fiber.New(),
	}

	// Register routes with dependencies
	router.RegisterRoutes(server.App, router.RouteDeps{
		DBService:           db,
		AuthService:         authService,
		BlogService:         blogService,
		ProjectsService:     projectsService,
		ImageService:        imageService,
		StorageService:      storageService,
		PagesService:        pagesService,
		SourcesService:      sourcesService,
		ChatService:         chatService,
		AgentCopilotMgr:     agentCopilotMgr,
		ProfileService:      profileService,
		OrganizationService: organizationService,
	})

	return server
}
