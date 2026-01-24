package handler

import (
	"blog-agent-go/backend/internal/errors"
	"blog-agent-go/backend/internal/response"
	"blog-agent-go/backend/internal/services"

	"github.com/gofiber/fiber/v2"
)

func ListFilesHandler(storageService *services.StorageService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		prefix := c.Query("prefix")
		files, folders, err := storageService.ListFiles(c.Context(), prefix)
		if err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, fiber.Map{
			"files":   files,
			"folders": folders,
		})
	}
}

func UploadFileHandler(storageService *services.StorageService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		file, err := c.FormFile("file")
		if err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid file upload"))
		}
		key := c.FormValue("key")
		if key == "" {
			return response.Error(c, errors.NewInvalidInputError("Key is required"))
		}
		data, err := file.Open()
		if err != nil {
			return response.Error(c, errors.NewInternalError("Failed to open file"))
		}
		defer data.Close()
		buf := make([]byte, file.Size)
		_, err = data.Read(buf)
		if err != nil {
			return response.Error(c, errors.NewInternalError("Failed to read file"))
		}
		if err := storageService.UploadFile(c.Context(), key, buf); err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, fiber.Map{"message": "File uploaded successfully"})
	}
}

func DeleteFileHandler(storageService *services.StorageService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := c.Params("key")
		if key == "" {
			return response.Error(c, errors.NewInvalidInputError("Key is required"))
		}
		if err := storageService.DeleteFile(c.Context(), key); err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, fiber.Map{"message": "File deleted successfully"})
	}
}

func CreateFolderHandler(storageService *services.StorageService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			Path string `json:"path"`
		}
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid request body"))
		}
		if err := storageService.CreateFolder(c.Context(), req.Path); err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, fiber.Map{"message": "Folder created successfully"})
	}
}

func UpdateFolderHandler(storageService *services.StorageService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			OldPath string `json:"old_path"`
			NewPath string `json:"new_path"`
		}
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid request body"))
		}
		if err := storageService.UpdateFolder(c.Context(), req.OldPath, req.NewPath); err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, fiber.Map{"message": "Folder updated successfully"})
	}
}
