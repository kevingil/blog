package dto

// CreateFolderRequest represents the request body for creating a folder
type CreateFolderRequest struct {
	Path string `json:"path" validate:"required"`
}

// UpdateFolderRequest represents the request body for updating a folder
type UpdateFolderRequest struct {
	OldPath string `json:"oldPath" validate:"required"`
	NewPath string `json:"newPath" validate:"required"`
}
