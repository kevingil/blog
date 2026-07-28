package dto

import "github.com/google/uuid"

// CreatePageRequest represents a request to create a page
type CreatePageRequest struct {
	Slug        string `json:"slug" validate:"required,slug"`
	Title       string `json:"title" validate:"required,min=1,max=200"`
	Content     string `json:"content"`
	IsPublished bool   `json:"is_published"`
}

// UpdatePageRequest represents a request to update a page
type UpdatePageRequest struct {
	Slug        *string `json:"slug" validate:"omitempty,slug"`
	Title       *string `json:"title" validate:"omitempty,min=1,max=200"`
	Content     *string `json:"content"`
	IsPublished *bool   `json:"is_published"`
}

// PageResponse represents a page in API responses
type PageResponse struct {
	ID          uuid.UUID `json:"id"`
	Slug        string    `json:"slug"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	IsPublished bool      `json:"is_published"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

// PageListResponse represents a paginated list of pages
type PageListResponse struct {
	Pages      []PageResponse `json:"pages"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PerPage    int            `json:"per_page"`
	TotalPages int            `json:"total_pages"`
}
