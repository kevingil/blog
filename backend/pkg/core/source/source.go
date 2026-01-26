// Package source provides the ArticleSource domain type and store interface
package source

import (
	"context"

	"backend/pkg/types"

	"github.com/google/uuid"
)

// Source is an alias to types.Source for backward compatibility
type Source = types.Source

// SourceStore defines the data access interface for article sources
type SourceStore interface {
	FindByID(ctx context.Context, id uuid.UUID) (*types.Source, error)
	FindByArticleID(ctx context.Context, articleID uuid.UUID) ([]types.Source, error)
	SearchSimilar(ctx context.Context, articleID uuid.UUID, embedding []float32, limit int) ([]types.Source, error)
	Save(ctx context.Context, source *types.Source) error
	Update(ctx context.Context, source *types.Source) error
	Delete(ctx context.Context, id uuid.UUID) error
}
