// Package page provides the Page domain type and store interface
package page

import (
	"context"

	"backend/pkg/types"

	"github.com/google/uuid"
)

// Page is an alias to types.Page for backward compatibility
type Page = types.Page

// ListOptions is an alias to types.PageListOptions for backward compatibility
type ListOptions = types.PageListOptions

// CreateRequest is an alias to types.PageCreateRequest for backward compatibility
type CreateRequest = types.PageCreateRequest

// UpdateRequest is an alias to types.PageUpdateRequest for backward compatibility
type UpdateRequest = types.PageUpdateRequest

// ListResult is an alias to types.PageListResult for backward compatibility
type ListResult = types.PageListResult

// PageStore defines the data access interface for pages
type PageStore interface {
	FindByID(ctx context.Context, id uuid.UUID) (*types.Page, error)
	FindBySlug(ctx context.Context, slug string) (*types.Page, error)
	List(ctx context.Context, opts types.PageListOptions) ([]types.Page, int64, error)
	Save(ctx context.Context, page *types.Page) error
	Delete(ctx context.Context, id uuid.UUID) error
}
