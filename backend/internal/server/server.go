package server

import (
	"blog-agent-go/backend/internal/database"
	"blog-agent-go/backend/internal/services"

	"github.com/gofiber/fiber/v2"
)

type FiberServer struct {
	App            *fiber.App
	db             database.Service
	authService    *services.AuthService
	blogService    *services.ArticleService
	imageService   *services.ImageGenerationService
	storageService *services.StorageService
	pagesService   *services.PagesService
}

func NewFiberServer(
	db database.Service,
	authService *services.AuthService,
	blogService *services.ArticleService,
	imageService *services.ImageGenerationService,
	storageService *services.StorageService,
	pagesService *services.PagesService,
) *FiberServer {
	server := &FiberServer{
		App:            fiber.New(),
		db:             db,
		authService:    authService,
		blogService:    blogService,
		imageService:   imageService,
		storageService: storageService,
		pagesService:   pagesService,
	}

	return server
}
