package dto

import "github.com/google/uuid"

// CreateSourceRequest represents a request to create a source
type CreateSourceRequest struct {
	ArticleID  uuid.UUID              `json:"article_id" validate:"required"`
	Title      string                 `json:"title" validate:"required,min=1,max=200"`
	Content    string                 `json:"content"`
	URL        string                 `json:"url" validate:"omitempty,url"`
	SourceType string                 `json:"source_type" validate:"omitempty,oneof=web pdf document note"`
	MetaData   map[string]interface{} `json:"meta_data"`
}

// UpdateSourceRequest represents a request to update a source
type UpdateSourceRequest struct {
	Title      *string                 `json:"title" validate:"omitempty,min=1,max=200"`
	Content    *string                 `json:"content"`
	URL        *string                 `json:"url" validate:"omitempty,url"`
	SourceType *string                 `json:"source_type" validate:"omitempty,oneof=web pdf document note"`
	MetaData   *map[string]interface{} `json:"meta_data"`
}

// SourceResponse represents a source in API responses
type SourceResponse struct {
	ID         uuid.UUID              `json:"id"`
	ArticleID  uuid.UUID              `json:"article_id"`
	Title      string                 `json:"title"`
	Content    string                 `json:"content"`
	URL        string                 `json:"url,omitempty"`
	SourceType string                 `json:"source_type"`
	MetaData   map[string]interface{} `json:"meta_data,omitempty"`
	CreatedAt  string                 `json:"created_at"`
	UpdatedAt  string                 `json:"updated_at"`
}
