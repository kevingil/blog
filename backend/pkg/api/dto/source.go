package dto

import (
	"backend/pkg/types"
	"time"

	"github.com/google/uuid"
)

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

// SourceWithArticleResponse is the list-endpoint DTO that flattens Source + article metadata
// and excludes the embedding vector.
type SourceWithArticleResponse struct {
	ID             uuid.UUID              `json:"id"`
	ArticleID      uuid.UUID              `json:"article_id"`
	Title          string                 `json:"title"`
	Content        string                 `json:"content"`
	ContentPreview string                 `json:"content_preview"`
	URL            string                 `json:"url,omitempty"`
	SourceType     string                 `json:"source_type"`
	MetaData       map[string]interface{} `json:"meta_data,omitempty"`
	CreatedAt      string                 `json:"created_at"`
	ArticleTitle   string                 `json:"article_title"`
	ArticleSlug    string                 `json:"article_slug"`
}

// SourceWithArticleToResponse converts a domain SourceWithArticle to the API response DTO,
// stripping the embedding and adding a content preview.
func SourceWithArticleToResponse(s types.SourceWithArticle) SourceWithArticleResponse {
	preview := s.Source.Content
	if len(preview) > 300 {
		preview = preview[:300] + "..."
	}

	return SourceWithArticleResponse{
		ID:             s.Source.ID,
		ArticleID:      s.Source.ArticleID,
		Title:          s.Source.Title,
		Content:        s.Source.Content,
		ContentPreview: preview,
		URL:            s.Source.URL,
		SourceType:     s.Source.SourceType,
		MetaData:       s.Source.MetaData,
		CreatedAt:      s.Source.CreatedAt.Format(time.RFC3339),
		ArticleTitle:   s.ArticleTitle,
		ArticleSlug:    s.ArticleSlug,
	}
}
