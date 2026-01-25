package types

import (
	"time"

	"github.com/google/uuid"
)

// Article represents a blog article with cached draft and published content
type Article struct {
	ID       uuid.UUID
	Slug     string
	AuthorID uuid.UUID
	TagIDs   []int64

	// Cached draft content (latest working version)
	DraftTitle     string
	DraftContent   string
	DraftImageURL  string
	DraftEmbedding []float32

	// Cached published content (nil if unpublished)
	PublishedTitle     *string
	PublishedContent   *string
	PublishedImageURL  *string
	PublishedEmbedding []float32
	PublishedAt        *time.Time

	// Version pointers (for history queries)
	CurrentDraftVersionID     *uuid.UUID
	CurrentPublishedVersionID *uuid.UUID

	// Metadata
	ImagenRequestID *uuid.UUID
	SessionMemory   map[string]interface{}
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// IsPublished returns true if the article has been published
func (a *Article) IsPublished() bool {
	return a.PublishedAt != nil
}

// ArticleVersion represents a historical snapshot of an article
type ArticleVersion struct {
	ID            uuid.UUID
	ArticleID     uuid.UUID
	VersionNumber int
	Status        string // "draft" or "published"
	Title         string
	Content       string
	ImageURL      string
	Embedding     []float32
	EditedBy      *uuid.UUID
	CreatedAt     time.Time
}

// ArticleSearchOptions represents options for searching articles
type ArticleSearchOptions struct {
	Query         string
	Page          int
	PerPage       int
	PublishedOnly bool
}

// ArticleListOptions represents options for listing articles
type ArticleListOptions struct {
	Page          int
	PerPage       int
	PublishedOnly bool
	AuthorID      *uuid.UUID
	TagID         *int
}
