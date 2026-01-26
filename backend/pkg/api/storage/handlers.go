package storage

import (
	"io"

	"backend/pkg/api/response"
	"backend/pkg/core"
	coreStorage "backend/pkg/core/storage"

	"github.com/gofiber/fiber/v2"
)

// ListFiles handles GET /storage/files
func ListFiles(c *fiber.Ctx) error {
	prefix := c.Query("prefix", "")

	result, err := coreStorage.ListFiles(c.Context(), prefix)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, result)
}

// UploadFile handles POST /storage/upload
func UploadFile(c *fiber.Ctx) error {
	// Get the key from form data
	key := c.FormValue("key")
	if key == "" {
		return response.Error(c, core.InvalidInputError("File key is required"))
	}

	// Get the file from form data
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return response.Error(c, core.InvalidInputError("File is required"))
	}

	// Open the file
	file, err := fileHeader.Open()
	if err != nil {
		return response.Error(c, err)
	}
	defer file.Close()

	// Read file contents
	data, err := io.ReadAll(file)
	if err != nil {
		return response.Error(c, err)
	}

	// Upload to storage
	if err := coreStorage.UploadFile(c.Context(), key, data); err != nil {
		return response.Error(c, err)
	}

	// Return the URL of the uploaded file
	url := coreStorage.GetURLPrefix() + "/" + key
	return response.Success(c, fiber.Map{
		"success": true,
		"url":     url,
		"key":     key,
	})
}

// DeleteFile handles DELETE /storage/:key
func DeleteFile(c *fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return response.Error(c, core.InvalidInputError("File key is required"))
	}

	if err := coreStorage.DeleteFile(c.Context(), key); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"success": true})
}

// CreateFolderRequest represents the request body for creating a folder
type CreateFolderRequest struct {
	Path string `json:"path"`
}

// CreateFolder handles POST /storage/folders
func CreateFolder(c *fiber.Ctx) error {
	var req CreateFolderRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}

	if req.Path == "" {
		return response.Error(c, core.InvalidInputError("Folder path is required"))
	}

	if err := coreStorage.CreateFolder(c.Context(), req.Path); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"success": true})
}

// UpdateFolderRequest represents the request body for updating a folder
type UpdateFolderRequest struct {
	OldPath string `json:"oldPath"`
	NewPath string `json:"newPath"`
}

// UpdateFolder handles PUT /storage/folders
func UpdateFolder(c *fiber.Ctx) error {
	var req UpdateFolderRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}

	if req.OldPath == "" {
		return response.Error(c, core.InvalidInputError("Old path is required"))
	}
	if req.NewPath == "" {
		return response.Error(c, core.InvalidInputError("New path is required"))
	}

	if err := coreStorage.UpdateFolder(c.Context(), req.OldPath, req.NewPath); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"success": true})
}
