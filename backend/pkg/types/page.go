package types

import (
	"time"

	"github.com/google/uuid"
)

// Page represents a static page in the blog
type Page struct {
	ID          uuid.UUID
	Slug        string
	Title       string
	Content     string
	Description string
	ImageURL    string
	MetaData    map[string]interface{}
	IsPublished bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// PageListOptions represents options for listing pages
type PageListOptions struct {
	Page        int
	PerPage     int
	IsPublished *bool
}

// PageCreateRequest represents a request to create a page
type PageCreateRequest struct {
	Slug        string                 `json:"slug" validate:"required,min=3,max=100"`
	Title       string                 `json:"title" validate:"required,min=3,max=200"`
	Content     string                 `json:"content" validate:"required,min=10"`
	Description string                 `json:"description" validate:"max=500"`
	ImageURL    string                 `json:"image_url" validate:"omitempty,url"`
	MetaData    map[string]interface{} `json:"meta_data"`
	IsPublished bool                   `json:"is_published"`
}

// PageUpdateRequest represents a request to update a page
type PageUpdateRequest struct {
	Title       *string                 `json:"title"`
	Content     *string                 `json:"content"`
	Description *string                 `json:"description"`
	ImageURL    *string                 `json:"image_url"`
	MetaData    *map[string]interface{} `json:"meta_data"`
	IsPublished *bool                   `json:"is_published"`
}

// PageListResult represents the result of listing pages
type PageListResult struct {
	Pages      []Page `json:"pages"`
	Total      int64  `json:"total"`
	Page       int    `json:"page"`
	PerPage    int    `json:"per_page"`
	TotalPages int    `json:"total_pages"`
}
