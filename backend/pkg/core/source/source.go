// Package source provides the ArticleSource domain type and store interface
package source

import (
	"context"

	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// Source is an alias to types.Source for backward compatibility
type Source = types.Source

// SourceListOptions represents options for listing sources
type SourceListOptions struct {
	Page    int
	PerPage int
}

// SourceWithArticle includes article metadata with the source
type SourceWithArticle struct {
	Source       types.Source
	ArticleTitle string
	ArticleSlug  string
}

// SourceStore defines the data access interface for article sources
type SourceStore interface {
	FindByID(ctx context.Context, id uuid.UUID) (*types.Source, error)
	FindByArticleID(ctx context.Context, articleID uuid.UUID) ([]types.Source, error)
	List(ctx context.Context, opts SourceListOptions) ([]SourceWithArticle, int64, error)
	SearchSimilar(ctx context.Context, articleID uuid.UUID, embedding []float32, limit int) ([]types.Source, error)
	Save(ctx context.Context, source *types.Source) error
	Update(ctx context.Context, source *types.Source) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ArticleStore defines the data access interface for articles (used for validation)
type ArticleStore interface {
	FindByID(ctx context.Context, id uuid.UUID) (*types.Article, error)
}

// EmbeddingService defines the interface for generating embeddings
type EmbeddingService interface {
	GenerateEmbedding(ctx context.Context, text string) (pgvector.Vector, error)
}
