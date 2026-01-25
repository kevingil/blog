package router

import (
	"backend/pkg/api/handler"
	"backend/pkg/api/middleware"

	"github.com/gofiber/fiber/v2"
)

// RegisterStorageRoutes registers file storage routes
func RegisterStorageRoutes(app *fiber.App, deps RouteDeps) {
	// All storage routes require authentication
	storage := app.Group("/storage", middleware.AuthMiddleware(deps.AuthService))

	storage.Get("/files", handler.ListFilesHandler(deps.StorageService))
	storage.Post("/upload", handler.UploadFileHandler(deps.StorageService))
	storage.Delete(":key", handler.DeleteFileHandler(deps.StorageService))
	storage.Post("/folders", handler.CreateFolderHandler(deps.StorageService))
	storage.Put("/folders", handler.UpdateFolderHandler(deps.StorageService))
}
