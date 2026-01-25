package dto

import "github.com/google/uuid"

// CreateOrganizationRequest represents a request to create an organization
type CreateOrganizationRequest struct {
	Name            string                 `json:"name" validate:"required,min=1,max=100"`
	Slug            string                 `json:"slug" validate:"omitempty,slug"`
	Bio             *string                `json:"bio" validate:"omitempty,max=500"`
	LogoURL         *string                `json:"logo_url" validate:"omitempty,url"`
	WebsiteURL      *string                `json:"website_url" validate:"omitempty,url"`
	EmailPublic     *string                `json:"email_public" validate:"omitempty,email"`
	SocialLinks     map[string]interface{} `json:"social_links"`
	MetaDescription *string                `json:"meta_description" validate:"omitempty,max=200"`
}

// UpdateOrganizationRequest represents a request to update an organization
type UpdateOrganizationRequest struct {
	Name            *string                 `json:"name" validate:"omitempty,min=1,max=100"`
	Slug            *string                 `json:"slug" validate:"omitempty,slug"`
	Bio             *string                 `json:"bio" validate:"omitempty,max=500"`
	LogoURL         *string                 `json:"logo_url" validate:"omitempty,url"`
	WebsiteURL      *string                 `json:"website_url" validate:"omitempty,url"`
	EmailPublic     *string                 `json:"email_public" validate:"omitempty,email"`
	SocialLinks     *map[string]interface{} `json:"social_links"`
	MetaDescription *string                 `json:"meta_description" validate:"omitempty,max=200"`
}

// OrganizationResponse represents an organization in API responses
type OrganizationResponse struct {
	ID              uuid.UUID              `json:"id"`
	Name            string                 `json:"name"`
	Slug            string                 `json:"slug"`
	Bio             *string                `json:"bio,omitempty"`
	LogoURL         *string                `json:"logo_url,omitempty"`
	WebsiteURL      *string                `json:"website_url,omitempty"`
	EmailPublic     *string                `json:"email_public,omitempty"`
	SocialLinks     map[string]interface{} `json:"social_links,omitempty"`
	MetaDescription *string                `json:"meta_description,omitempty"`
	CreatedAt       string                 `json:"created_at"`
	UpdatedAt       string                 `json:"updated_at"`
}

// JoinOrganizationRequest represents a request to join an organization
type JoinOrganizationRequest struct {
	OrganizationID uuid.UUID `json:"organization_id" validate:"required"`
}
