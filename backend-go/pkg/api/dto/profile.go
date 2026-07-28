package dto

import "github.com/google/uuid"

// ProfileUpdateRequest is the request to update a user profile
type ProfileUpdateRequest struct {
	Name            *string            `json:"name"`
	Bio             *string            `json:"bio"`
	ProfileImage    *string            `json:"profile_image"`
	EmailPublic     *string            `json:"email_public"`
	SocialLinks     *map[string]string `json:"social_links"`
	MetaDescription *string            `json:"meta_description"`
}

// PublicProfileResponse is the public profile response
type PublicProfileResponse struct {
	Type            string            `json:"type"` // "user" or "organization"
	ID              uuid.UUID         `json:"id"`
	Name            string            `json:"name"`
	Bio             string            `json:"bio"`
	ImageURL        string            `json:"image_url"` // profile_image for user, logo_url for org
	EmailPublic     string            `json:"email_public"`
	SocialLinks     map[string]string `json:"social_links"`
	MetaDescription string            `json:"meta_description"`
	WebsiteURL      *string           `json:"website_url,omitempty"` // org only
}

// SiteSettingsResponse is the response for site settings
type SiteSettingsResponse struct {
	PublicProfileType    string     `json:"public_profile_type"`
	PublicUserID         *uuid.UUID `json:"public_user_id"`
	PublicOrganizationID *uuid.UUID `json:"public_organization_id"`
}

// SiteSettingsUpdateRequest is the request to update site settings
type SiteSettingsUpdateRequest struct {
	PublicProfileType    *string    `json:"public_profile_type"`
	PublicUserID         *uuid.UUID `json:"public_user_id"`
	PublicOrganizationID *uuid.UUID `json:"public_organization_id"`
}

// UserProfileResponse is the profile data response for a user account
type UserProfileResponse struct {
	ID              uuid.UUID         `json:"id"`
	Name            string            `json:"name"`
	Bio             string            `json:"bio"`
	ProfileImage    string            `json:"profile_image"`
	EmailPublic     string            `json:"email_public"`
	SocialLinks     map[string]string `json:"social_links"`
	MetaDescription string            `json:"meta_description"`
	OrganizationID  *uuid.UUID        `json:"organization_id,omitempty"`
}
