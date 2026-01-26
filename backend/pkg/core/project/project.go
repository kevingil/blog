// Package project provides the Project domain type and store interface
package project

import (
	"context"

	"backend/pkg/types"

	"github.com/google/uuid"
)

// Project is an alias to types.Project for backward compatibility
type Project = types.Project

// ListOptions is an alias to types.ProjectListOptions for backward compatibility
type ListOptions = types.ProjectListOptions

// ProjectStore defines the data access interface for projects
type ProjectStore interface {
	FindByID(ctx context.Context, id uuid.UUID) (*types.Project, error)
	List(ctx context.Context, opts types.ProjectListOptions) ([]types.Project, int64, error)
	Save(ctx context.Context, project *types.Project) error
	Update(ctx context.Context, project *types.Project) error
	Delete(ctx context.Context, id uuid.UUID) error
}
