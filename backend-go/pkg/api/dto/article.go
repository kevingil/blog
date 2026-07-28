// Package dto provides Data Transfer Objects for API requests and responses
package dto

import "github.com/google/uuid"

// CreateArticleRequest represents a request to create an article
type CreateArticleRequest struct {
	Title    string   `json:"title" validate:"required,min=1,max=200"`
	Content  string   `json:"content" validate:"required"`
	Slug     string   `json:"slug" validate:"omitempty,slug"`
	Tags     []string `json:"tags" validate:"omitempty,dive,min=1,max=50"`
	Publish  bool     `json:"publish"`
	ImageURL string   `json:"image_url" validate:"omitempty,url"`
}

// UpdateArticleRequest represents a request to update an article (updates draft)
type UpdateArticleRequest struct {
	Title    *string   `json:"title" validate:"omitempty,min=1,max=200"`
	Content  *string   `json:"content"`
	Slug     *string   `json:"slug" validate:"omitempty,slug"`
	Tags     *[]string `json:"tags" validate:"omitempty,dive,min=1,max=50"`
	ImageURL *string   `json:"image_url" validate:"omitempty,url"`
}

// ArticleResponse represents an article in API responses
type ArticleResponse struct {
	ID uuid.UUID `json:"id"`

	// Common fields
	Slug string   `json:"slug"`
	Tags []string `json:"tags,omitempty"`

	// Draft content (for editing)
	DraftTitle    string `json:"draft_title"`
	DraftContent  string `json:"draft_content"`
	DraftImageURL string `json:"draft_image_url,omitempty"`

	// Published content (for public viewing)
	PublishedTitle    *string `json:"published_title,omitempty"`
	PublishedContent  *string `json:"published_content,omitempty"`
	PublishedImageURL *string `json:"published_image_url,omitempty"`
	PublishedAt       *string `json:"published_at,omitempty"`

	// Metadata
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ArticleListResponse represents a paginated list of articles
type ArticleListResponse struct {
	Articles   []ArticleResponse `json:"articles"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	PerPage    int               `json:"per_page"`
	TotalPages int               `json:"total_pages"`
}

// ArticleVersionResponse represents a version in API responses
type ArticleVersionResponse struct {
	ID            uuid.UUID `json:"id"`
	ArticleID     uuid.UUID `json:"article_id"`
	VersionNumber int       `json:"version_number"`
	Status        string    `json:"status"`
	Title         string    `json:"title"`
	Content       string    `json:"content"`
	ImageURL      string    `json:"image_url,omitempty"`
	CreatedAt     string    `json:"created_at"`
}

// ArticleVersionListResponse represents a list of versions
type ArticleVersionListResponse struct {
	Versions []ArticleVersionResponse `json:"versions"`
	Total    int                      `json:"total"`
}
