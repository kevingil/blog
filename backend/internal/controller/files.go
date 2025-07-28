package controller

import (
	"blog-agent-go/backend/internal/services"

	"github.com/gofiber/fiber/v2"
)

func ListFilesHandler(storageService *services.StorageService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		prefix := c.Query("prefix")
		files, folders, err := storageService.ListFiles(c.Context(), prefix)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{
			"files":   files,
			"folders": folders,
		})
	}
}

func UploadFileHandler(storageService *services.StorageService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		file, err := c.FormFile("file")
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid file upload"})
		}
		key := c.FormValue("key")
		if key == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Key is required"})
		}
		data, err := file.Open()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		defer data.Close()
		buf := make([]byte, file.Size)
		_, err = data.Read(buf)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		if err := storageService.UploadFile(c.Context(), key, buf); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "File uploaded successfully"})
	}
}

func DeleteFileHandler(storageService *services.StorageService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := c.Params("key")
		if key == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Key is required"})
		}
		if err := storageService.DeleteFile(c.Context(), key); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "File deleted successfully"})
	}
}

func CreateFolderHandler(storageService *services.StorageService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			Path string `json:"path"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		if err := storageService.CreateFolder(c.Context(), req.Path); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "Folder created successfully"})
	}
}

func UpdateFolderHandler(storageService *services.StorageService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			OldPath string `json:"old_path"`
			NewPath string `json:"new_path"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		if err := storageService.UpdateFolder(c.Context(), req.OldPath, req.NewPath); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "Folder updated successfully"})
	}
}
