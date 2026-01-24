// Package project provides the Project domain type and store interface
package project

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Project represents a portfolio project
type Project struct {
	ID          uuid.UUID
	Title       string
	Description string
	Content     string
	TagIDs      []int64
	ImageURL    string
	URL         string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ListOptions represents options for listing projects
type ListOptions struct {
	Page    int
	PerPage int
}

// ProjectStore defines the data access interface for projects
type ProjectStore interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Project, error)
	List(ctx context.Context, opts ListOptions) ([]Project, int64, error)
	Save(ctx context.Context, project *Project) error
	Update(ctx context.Context, project *Project) error
	Delete(ctx context.Context, id uuid.UUID) error
}
