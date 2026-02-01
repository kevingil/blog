package storage

import (
	"io"

	"backend/pkg/api/dto"
	"backend/pkg/api/response"
	"backend/pkg/api/validation"
	"backend/pkg/core"
	coreStorage "backend/pkg/core/storage"

	"github.com/gofiber/fiber/v2"
)

// ListFiles handles GET /storage/files
// @Summary List files
// @Description Get a list of files in storage
// @Tags storage
// @Accept json
// @Produce json
// @Param prefix query string false "File prefix filter"
// @Success 200 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /storage/files [get]
func ListFiles(c *fiber.Ctx) error {
	prefix := c.Query("prefix", "")

	result, err := coreStorage.ListFiles(c.Context(), prefix)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, result)
}

// UploadFile handles POST /storage/upload
// @Summary Upload file
// @Description Upload a file to storage
// @Tags storage
// @Accept multipart/form-data
// @Produce json
// @Param key formData string true "File key/path"
// @Param file formData file true "File to upload"
// @Success 200 {object} response.SuccessResponse{data=object{success=boolean,url=string,key=string}}
// @Failure 400 {object} response.SuccessResponse
// @Failure 401 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /storage/upload [post]
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
// @Summary Delete file
// @Description Delete a file from storage
// @Tags storage
// @Accept json
// @Produce json
// @Param key path string true "File key/path"
// @Success 200 {object} response.SuccessResponse{data=object{success=boolean}}
// @Failure 400 {object} response.SuccessResponse
// @Failure 401 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /storage/{key} [delete]
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

// CreateFolder handles POST /storage/folders
// @Summary Create folder
// @Description Create a new folder in storage
// @Tags storage
// @Accept json
// @Produce json
// @Param request body dto.CreateFolderRequest true "Folder path"
// @Success 200 {object} response.SuccessResponse{data=object{success=boolean}}
// @Failure 400 {object} response.SuccessResponse
// @Failure 401 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /storage/folders [post]
func CreateFolder(c *fiber.Ctx) error {
	var req dto.CreateFolderRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}
	if err := validation.ValidateStruct(req); err != nil {
		return response.Error(c, err)
	}

	if err := coreStorage.CreateFolder(c.Context(), req.Path); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"success": true})
}

// UpdateFolder handles PUT /storage/folders
// @Summary Update folder
// @Description Rename/move a folder in storage
// @Tags storage
// @Accept json
// @Produce json
// @Param request body dto.UpdateFolderRequest true "Old and new folder paths"
// @Success 200 {object} response.SuccessResponse{data=object{success=boolean}}
// @Failure 400 {object} response.SuccessResponse
// @Failure 401 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /storage/folders [put]
func UpdateFolder(c *fiber.Ctx) error {
	var req dto.UpdateFolderRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}
	if err := validation.ValidateStruct(req); err != nil {
		return response.Error(c, err)
	}

	if err := coreStorage.UpdateFolder(c.Context(), req.OldPath, req.NewPath); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"success": true})
}
