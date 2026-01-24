// Package article provides the Article domain type and store interface
package article

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Article represents a blog article
type Article struct {
	ID              uuid.UUID
	Slug            string
	Title           string
	Content         string
	ImageURL        string
	AuthorID        uuid.UUID
	TagIDs          []int64
	IsDraft         bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
	PublishedAt     *time.Time
	ImagenRequestID *uuid.UUID
	Embedding       []float32
	SessionMemory   map[string]interface{}
}

// SearchOptions represents options for searching articles
type SearchOptions struct {
	Query   string
	Page    int
	PerPage int
	IsDraft *bool
}

// ListOptions represents options for listing articles
type ListOptions struct {
	Page     int
	PerPage  int
	IsDraft  *bool
	AuthorID *uuid.UUID
	TagID    *int
}

// ArticleStore defines the data access interface for articles
type ArticleStore interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Article, error)
	FindBySlug(ctx context.Context, slug string) (*Article, error)
	List(ctx context.Context, opts ListOptions) ([]Article, int64, error)
	Search(ctx context.Context, opts SearchOptions) ([]Article, int64, error)
	SearchByEmbedding(ctx context.Context, embedding []float32, limit int) ([]Article, error)
	Save(ctx context.Context, article *Article) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetPopularTags(ctx context.Context, limit int) ([]int64, error)
}
