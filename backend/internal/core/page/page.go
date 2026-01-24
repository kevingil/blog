// Package page provides the Page domain type and store interface
package page

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Page represents a static page in the blog
type Page struct {
	ID          uuid.UUID
	Slug        string
	Title       string
	Content     string
	Description string
	ImageURL    string
	MetaData    map[string]interface{}
	IsPublished bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ListOptions represents options for listing pages
type ListOptions struct {
	Page    int
	PerPage int
}

// PageStore defines the data access interface for pages
type PageStore interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Page, error)
	FindBySlug(ctx context.Context, slug string) (*Page, error)
	List(ctx context.Context, opts ListOptions) ([]Page, int64, error)
	Save(ctx context.Context, page *Page) error
	Delete(ctx context.Context, id uuid.UUID) error
}
