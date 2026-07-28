// Package source provides the ArticleSource domain type
package source

import (
	"context"

	"backend/pkg/types"

	"github.com/pgvector/pgvector-go"
)

// Source is an alias to types.Source for backward compatibility
type Source = types.Source

// SourceListOptions is an alias to types.SourceListOptions for backward compatibility
type SourceListOptions = types.SourceListOptions

// SourceWithArticle is an alias to types.SourceWithArticle for backward compatibility
type SourceWithArticle = types.SourceWithArticle

// EmbeddingService defines the interface for generating embeddings
type EmbeddingService interface {
	GenerateEmbedding(ctx context.Context, text string) (pgvector.Vector, error)
}
