package storage

import (
	"io"
	"log"

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
	log.Printf("[Storage] ListFiles: prefix=%q", prefix)

	result, err := coreStorage.ListFiles(c.Context(), prefix)
	if err != nil {
		log.Printf("[Storage] ListFiles: error: %v", err)
		return response.Error(c, err)
	}
	log.Printf("[Storage] ListFiles: prefix=%q files=%d folders=%d", prefix, len(result.Files), len(result.Folders))
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
	contentType := c.Get("Content-Type")
	log.Printf("[Storage] UploadFile: content-type=%q method=%s", contentType, c.Method())

	// Get the key from form data
	key := c.FormValue("key")
	log.Printf("[Storage] UploadFile: key=%q", key)
	if key == "" {
		log.Printf("[Storage] UploadFile: key is empty, returning 400")
		return response.Error(c, core.InvalidInputError("File key is required"))
	}

	// Get the file from form data
	fileHeader, err := c.FormFile("file")
	if err != nil {
		log.Printf("[Storage] UploadFile: failed to get file from form: %v", err)
		return response.Error(c, core.InvalidInputError("File is required"))
	}
	log.Printf("[Storage] UploadFile: file=%q size=%d", fileHeader.Filename, fileHeader.Size)

	// Open the file
	file, err := fileHeader.Open()
	if err != nil {
		log.Printf("[Storage] UploadFile: failed to open file: %v", err)
		return response.Error(c, err)
	}
	defer file.Close()

	// Read file contents
	data, err := io.ReadAll(file)
	if err != nil {
		log.Printf("[Storage] UploadFile: failed to read file: %v", err)
		return response.Error(c, err)
	}

	// Upload to storage
	log.Printf("[Storage] UploadFile: uploading key=%q (%d bytes)", key, len(data))
	if err := coreStorage.UploadFile(c.Context(), key, data); err != nil {
		log.Printf("[Storage] UploadFile: storage upload failed: %v", err)
		return response.Error(c, err)
	}

	// Return the URL of the uploaded file
	url := coreStorage.GetURLPrefix() + "/" + key
	log.Printf("[Storage] UploadFile: success key=%q url=%q", key, url)
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
	log.Printf("[Storage] DeleteFile: key=%q", key)
	if key == "" {
		return response.Error(c, core.InvalidInputError("File key is required"))
	}

	if err := coreStorage.DeleteFile(c.Context(), key); err != nil {
		log.Printf("[Storage] DeleteFile: error key=%q: %v", key, err)
		return response.Error(c, err)
	}
	log.Printf("[Storage] DeleteFile: success key=%q", key)
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
		log.Printf("[Storage] CreateFolder: failed to parse body: %v", err)
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}
	if err := validation.ValidateStruct(req); err != nil {
		log.Printf("[Storage] CreateFolder: validation error: %v", err)
		return response.Error(c, err)
	}

	log.Printf("[Storage] CreateFolder: path=%q", req.Path)
	if err := coreStorage.CreateFolder(c.Context(), req.Path); err != nil {
		log.Printf("[Storage] CreateFolder: error path=%q: %v", req.Path, err)
		return response.Error(c, err)
	}
	log.Printf("[Storage] CreateFolder: success path=%q", req.Path)
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
