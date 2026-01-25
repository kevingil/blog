// Package dto provides Data Transfer Objects for API requests and responses
package dto

import "github.com/google/uuid"

// CreateArticleRequest represents a request to create an article
type CreateArticleRequest struct {
	Title    string   `json:"title" validate:"required,min=1,max=200"`
	Content  string   `json:"content" validate:"required"`
	Slug     string   `json:"slug" validate:"omitempty,slug"`
	Tags     []string `json:"tags" validate:"omitempty,dive,min=1,max=50"`
	IsDraft  bool     `json:"is_draft"`
	ImageURL string   `json:"image_url" validate:"omitempty,url"`
}

// UpdateArticleRequest represents a request to update an article
type UpdateArticleRequest struct {
	Title    *string   `json:"title" validate:"omitempty,min=1,max=200"`
	Content  *string   `json:"content"`
	Slug     *string   `json:"slug" validate:"omitempty,slug"`
	Tags     *[]string `json:"tags" validate:"omitempty,dive,min=1,max=50"`
	IsDraft  *bool     `json:"is_draft"`
	ImageURL *string   `json:"image_url" validate:"omitempty,url"`
}

// ArticleResponse represents an article in API responses
type ArticleResponse struct {
	ID          uuid.UUID `json:"id"`
	Slug        string    `json:"slug"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	ImageURL    string    `json:"image_url,omitempty"`
	IsDraft     bool      `json:"is_draft"`
	Tags        []string  `json:"tags,omitempty"`
	PublishedAt *string   `json:"published_at,omitempty"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

// ArticleListResponse represents a paginated list of articles
type ArticleListResponse struct {
	Articles   []ArticleResponse `json:"articles"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	PerPage    int               `json:"per_page"`
	TotalPages int               `json:"total_pages"`
}
