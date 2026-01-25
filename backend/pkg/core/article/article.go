// Package article provides the Article domain type and store interface
package article

import (
	"context"
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

// SearchOptions represents options for searching articles
type SearchOptions struct {
	Query         string
	Page          int
	PerPage       int
	PublishedOnly bool
}

// ListOptions represents options for listing articles
type ListOptions struct {
	Page          int
	PerPage       int
	PublishedOnly bool
	AuthorID      *uuid.UUID
	TagID         *int
}

// ArticleStore defines the data access interface for articles
type ArticleStore interface {
	// Basic CRUD operations
	FindByID(ctx context.Context, id uuid.UUID) (*Article, error)
	FindBySlug(ctx context.Context, slug string) (*Article, error)
	List(ctx context.Context, opts ListOptions) ([]Article, int64, error)
	Search(ctx context.Context, opts SearchOptions) ([]Article, int64, error)
	SearchByEmbedding(ctx context.Context, embedding []float32, limit int) ([]Article, error)
	Save(ctx context.Context, article *Article) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetPopularTags(ctx context.Context, limit int) ([]int64, error)

	// Version management operations
	SaveDraft(ctx context.Context, article *Article) error
	Publish(ctx context.Context, article *Article) error
	Unpublish(ctx context.Context, article *Article) error
	ListVersions(ctx context.Context, articleID uuid.UUID) ([]ArticleVersion, error)
	GetVersion(ctx context.Context, versionID uuid.UUID) (*ArticleVersion, error)
	RevertToVersion(ctx context.Context, articleID, versionID uuid.UUID) error
}
