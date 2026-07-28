package types

import (
	"time"

	"github.com/google/uuid"
)

// SiteSettings represents the site-wide settings
type SiteSettings struct {
	ID                   int
	PublicProfileType    string // 'user' or 'organization'
	PublicUserID         *uuid.UUID
	PublicOrganizationID *uuid.UUID
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// PublicProfile represents the public-facing profile (either user or organization)
type PublicProfile struct {
	Type            string // 'user' or 'organization'
	Name            string
	Bio             *string
	ImageURL        *string
	WebsiteURL      *string
	EmailPublic     *string
	SocialLinks     map[string]interface{}
	MetaDescription *string
}

// UserProfile is the domain model for a user account profile
type UserProfile struct {
	ID              uuid.UUID
	Name            string
	Bio             string
	ProfileImage    string
	EmailPublic     string
	SocialLinks     map[string]string
	MetaDescription string
	OrganizationID  *uuid.UUID
}
