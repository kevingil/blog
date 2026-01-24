// Package source provides the ArticleSource domain type and store interface
package source

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Source represents a source/citation for an article
type Source struct {
	ID         uuid.UUID
	ArticleID  uuid.UUID
	Title      string
	Content    string
	URL        string
	SourceType string
	Embedding  []float32
	MetaData   map[string]interface{}
	CreatedAt  time.Time
}

// SourceStore defines the data access interface for article sources
type SourceStore interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Source, error)
	FindByArticleID(ctx context.Context, articleID uuid.UUID) ([]Source, error)
	SearchSimilar(ctx context.Context, articleID uuid.UUID, embedding []float32, limit int) ([]Source, error)
	Save(ctx context.Context, source *Source) error
	Update(ctx context.Context, source *Source) error
	Delete(ctx context.Context, id uuid.UUID) error
}
