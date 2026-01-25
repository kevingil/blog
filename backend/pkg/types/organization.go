package types

import (
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

// OrganizationCreateRequest represents a request to create an organization
type OrganizationCreateRequest struct {
	Name            string             `json:"name" validate:"required,min=2,max=255"`
	Slug            string             `json:"slug" validate:"omitempty,min=2,max=100"`
	Bio             *string            `json:"bio"`
	LogoURL         *string            `json:"logo_url"`
	WebsiteURL      *string            `json:"website_url"`
	EmailPublic     *string            `json:"email_public"`
	SocialLinks     *map[string]string `json:"social_links"`
	MetaDescription *string            `json:"meta_description"`
}

// OrganizationUpdateRequest represents a request to update an organization
type OrganizationUpdateRequest struct {
	Name            *string            `json:"name"`
	Slug            *string            `json:"slug"`
	Bio             *string            `json:"bio"`
	LogoURL         *string            `json:"logo_url"`
	WebsiteURL      *string            `json:"website_url"`
	EmailPublic     *string            `json:"email_public"`
	SocialLinks     *map[string]string `json:"social_links"`
	MetaDescription *string            `json:"meta_description"`
}

// OrganizationResponse is the response for an organization
type OrganizationResponse struct {
	ID              uuid.UUID         `json:"id"`
	Name            string            `json:"name"`
	Slug            string            `json:"slug"`
	Bio             string            `json:"bio"`
	LogoURL         string            `json:"logo_url"`
	WebsiteURL      string            `json:"website_url"`
	EmailPublic     string            `json:"email_public"`
	SocialLinks     map[string]string `json:"social_links"`
	MetaDescription string            `json:"meta_description"`
}
