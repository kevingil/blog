// Package organization provides the Organization domain type and store interface
package organization

import (
	"context"

	"backend/pkg/types"

	"github.com/google/uuid"
)

// Organization is an alias to types.Organization for backward compatibility
type Organization = types.Organization

// CreateRequest is an alias to types.OrganizationCreateRequest for backward compatibility
type CreateRequest = types.OrganizationCreateRequest

// UpdateRequest is an alias to types.OrganizationUpdateRequest for backward compatibility
type UpdateRequest = types.OrganizationUpdateRequest

// OrganizationResponse is an alias to types.OrganizationResponse for backward compatibility
type OrganizationResponse = types.OrganizationResponse

// OrganizationStore defines the data access interface for organizations
type OrganizationStore interface {
	FindByID(ctx context.Context, id uuid.UUID) (*types.Organization, error)
	FindBySlug(ctx context.Context, slug string) (*types.Organization, error)
	List(ctx context.Context) ([]types.Organization, error)
	Save(ctx context.Context, org *types.Organization) error
	Update(ctx context.Context, org *types.Organization) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// AccountStore defines the data access interface for accounts (used for join/leave organization)
type AccountStore interface {
	FindByID(ctx context.Context, id uuid.UUID) (*types.Account, error)
	Update(ctx context.Context, account *types.Account) error
}
