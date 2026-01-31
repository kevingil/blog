// Package article provides the Article domain type and store interface
package article

import (
	"context"

	"backend/pkg/types"

	"github.com/google/uuid"
)

// Article is an alias to types.Article for backward compatibility
type Article = types.Article

// ArticleVersion is an alias to types.ArticleVersion for backward compatibility
type ArticleVersion = types.ArticleVersion

// SearchOptions is an alias to types.ArticleSearchOptions for backward compatibility
type SearchOptions = types.ArticleSearchOptions

// ListOptions is an alias to types.ArticleListOptions for backward compatibility
type ListOptions = types.ArticleListOptions

// ArticleStore defines the data access interface for articles
type ArticleStore interface {
	// Basic CRUD operations
	FindByID(ctx context.Context, id uuid.UUID) (*types.Article, error)
	FindBySlug(ctx context.Context, slug string) (*types.Article, error)
	List(ctx context.Context, opts types.ArticleListOptions) ([]types.Article, int64, error)
	Search(ctx context.Context, opts types.ArticleSearchOptions) ([]types.Article, int64, error)
	SearchByEmbedding(ctx context.Context, embedding []float32, limit int) ([]types.Article, error)
	Save(ctx context.Context, article *types.Article) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetPopularTags(ctx context.Context, limit int) ([]int64, error)

	// Slug uniqueness check
	SlugExists(ctx context.Context, slug string, excludeID *uuid.UUID) (bool, error)

	// Version management operations
	SaveDraft(ctx context.Context, article *types.Article) error
	Publish(ctx context.Context, article *types.Article) error
	Unpublish(ctx context.Context, article *types.Article) error
	ListVersions(ctx context.Context, articleID uuid.UUID) ([]types.ArticleVersion, error)
	GetVersion(ctx context.Context, versionID uuid.UUID) (*types.ArticleVersion, error)
	RevertToVersion(ctx context.Context, articleID, versionID uuid.UUID) error
}

// AccountStore defines the data access interface for accounts (used for author lookup)
type AccountStore interface {
	FindByID(ctx context.Context, id uuid.UUID) (*types.Account, error)
}

// TagStore defines the data access interface for tags
type TagStore interface {
	FindByID(ctx context.Context, id int) (*types.Tag, error)
	FindByName(ctx context.Context, name string) (*types.Tag, error)
	FindByIDs(ctx context.Context, ids []int64) ([]types.Tag, error)
	EnsureExists(ctx context.Context, names []string) ([]int64, error)
	List(ctx context.Context) ([]types.Tag, error)
	Save(ctx context.Context, tag *types.Tag) error
	Delete(ctx context.Context, id int) error
}
