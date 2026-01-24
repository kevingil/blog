// Package tag provides the Tag domain type and store interface
package tag

import (
	"context"
	"time"
)

// Tag represents a tag for categorizing articles and projects
type Tag struct {
	ID        int
	Name      string
	CreatedAt time.Time
}

// TagStore defines the data access interface for tags
type TagStore interface {
	FindByID(ctx context.Context, id int) (*Tag, error)
	FindByName(ctx context.Context, name string) (*Tag, error)
	FindByIDs(ctx context.Context, ids []int64) ([]Tag, error)
	EnsureExists(ctx context.Context, names []string) ([]int64, error)
	List(ctx context.Context) ([]Tag, error)
	Save(ctx context.Context, tag *Tag) error
	Delete(ctx context.Context, id int) error
}
