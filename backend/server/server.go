package server

import (
	"blog-agent-go/backend/services/auth"
	"blog-agent-go/backend/services/blog"
	"blog-agent-go/backend/services/images"
	"blog-agent-go/backend/services/pages"
	"blog-agent-go/backend/services/storage"
	"blog-agent-go/backend/services/user"

	"gorm.io/gorm"

	"github.com/gofiber/fiber/v2"
)

type FiberServer struct {
	App            *fiber.App
	db             *gorm.DB
	authService    *auth.AuthService
	userService    *user.AuthService
	blogService    *blog.ArticleService
	imageService   *images.ImageGenerationService
	storageService *storage.StorageService
	pagesService   *pages.Service
}

func NewFiberServer(
	db *gorm.DB,
	authService *auth.AuthService,
	userService *user.AuthService,
	blogService *blog.ArticleService,
	imageService *images.ImageGenerationService,
	storageService *storage.StorageService,
	pagesService *pages.Service,
) *FiberServer {
	server := &FiberServer{
		App:            fiber.New(),
		db:             db,
		authService:    authService,
		userService:    userService,
		blogService:    blogService,
		imageService:   imageService,
		storageService: storageService,
		pagesService:   pagesService,
	}

	server.RegisterFiberRoutes()

	return server
}
