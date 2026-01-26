// Package tag provides the Tag domain type and store interface
package tag

import (
	"context"

	"backend/pkg/types"
)

// Tag is an alias to types.Tag for backward compatibility
type Tag = types.Tag

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
