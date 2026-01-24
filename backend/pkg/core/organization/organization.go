// Package organization provides the Organization domain type and store interface
package organization

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Organization represents an organization profile
type Organization struct {
	ID              uuid.UUID
	Name            string
	Slug            string
	Bio             *string
	LogoURL         *string
	WebsiteURL      *string
	EmailPublic     *string
	SocialLinks     map[string]interface{}
	MetaDescription *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// OrganizationStore defines the data access interface for organizations
type OrganizationStore interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Organization, error)
	FindBySlug(ctx context.Context, slug string) (*Organization, error)
	List(ctx context.Context) ([]Organization, error)
	Save(ctx context.Context, org *Organization) error
	Update(ctx context.Context, org *Organization) error
	Delete(ctx context.Context, id uuid.UUID) error
}
