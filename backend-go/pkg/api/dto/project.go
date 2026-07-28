package dto

import "github.com/google/uuid"

// CreateProjectRequest represents a request to create a project
type CreateProjectRequest struct {
	Title       string   `json:"title" validate:"required,min=1,max=200"`
	Description string   `json:"description" validate:"required,min=1,max=500"`
	Content     string   `json:"content"`
	Tags        []string `json:"tags" validate:"omitempty,dive,min=1,max=50"`
	ImageURL    string   `json:"image_url" validate:"omitempty,url"`
	URL         string   `json:"url" validate:"omitempty,url"`
}

// UpdateProjectRequest represents a request to update a project
type UpdateProjectRequest struct {
	Title       *string   `json:"title" validate:"omitempty,min=1,max=200"`
	Description *string   `json:"description" validate:"omitempty,min=1,max=500"`
	Content     *string   `json:"content"`
	Tags        *[]string `json:"tags" validate:"omitempty,dive,min=1,max=50"`
	ImageURL    *string   `json:"image_url" validate:"omitempty,url"`
	URL         *string   `json:"url" validate:"omitempty,url"`
}

// ProjectResponse represents a project in API responses
type ProjectResponse struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	Tags        []string  `json:"tags,omitempty"`
	ImageURL    string    `json:"image_url,omitempty"`
	URL         string    `json:"url,omitempty"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

// ProjectListResponse represents a paginated list of projects
type ProjectListResponse struct {
	Projects   []ProjectResponse `json:"projects"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	PerPage    int               `json:"per_page"`
	TotalPages int               `json:"total_pages"`
}
