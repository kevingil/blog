// Package profile provides profile and site settings domain types and store interfaces
package profile

import (
	"context"
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

// SiteSettingsStore defines the data access interface for site settings
type SiteSettingsStore interface {
	Get(ctx context.Context) (*SiteSettings, error)
	Save(ctx context.Context, settings *SiteSettings) error
}

// ProfileStore defines the data access interface for profile operations
type ProfileStore interface {
	GetPublicProfile(ctx context.Context) (*PublicProfile, error)
	IsUserAdmin(ctx context.Context, userID uuid.UUID) (bool, error)
}
