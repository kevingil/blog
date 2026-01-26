// Package storage provides HTTP handlers for file storage management
package storage

import (
	"backend/pkg/api/middleware"

	"github.com/gofiber/fiber/v2"
)

// Register registers storage routes on the app
func Register(app *fiber.App) {
	// All storage routes require authentication
	storage := app.Group("/storage", middleware.Auth())
	storage.Get("/files", ListFiles)
	storage.Post("/upload", UploadFile)
	storage.Delete("/:key", DeleteFile)
	storage.Post("/folders", CreateFolder)
	storage.Put("/folders", UpdateFolder)
}
