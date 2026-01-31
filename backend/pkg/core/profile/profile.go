// Package profile provides profile and site settings domain types and store interfaces
package profile

import (
	"context"

	"backend/pkg/types"

	"github.com/google/uuid"
)

// SiteSettings is an alias to types.SiteSettings for backward compatibility
type SiteSettings = types.SiteSettings

// PublicProfile is an alias to types.PublicProfile for backward compatibility
type PublicProfile = types.PublicProfile

// UserProfile is an alias to types.UserProfile for backward compatibility
type UserProfile = types.UserProfile

// ProfileUpdateRequest is an alias to types.ProfileUpdateRequest for backward compatibility
type ProfileUpdateRequest = types.ProfileUpdateRequest

// PublicProfileResponse is an alias to types.PublicProfileResponse for backward compatibility
type PublicProfileResponse = types.PublicProfileResponse

// SiteSettingsResponse is an alias to types.SiteSettingsResponse for backward compatibility
type SiteSettingsResponse = types.SiteSettingsResponse

// SiteSettingsUpdateRequest is an alias to types.SiteSettingsUpdateRequest for backward compatibility
type SiteSettingsUpdateRequest = types.SiteSettingsUpdateRequest

// SiteSettingsStore defines the data access interface for site settings
type SiteSettingsStore interface {
	Get(ctx context.Context) (*types.SiteSettings, error)
	Save(ctx context.Context, settings *types.SiteSettings) error
}

// ProfileStore defines the data access interface for profile operations
type ProfileStore interface {
	GetPublicProfile(ctx context.Context) (*types.PublicProfile, error)
	IsUserAdmin(ctx context.Context, userID uuid.UUID) (bool, error)
}

// AccountStore defines the data access interface for account operations
type AccountStore interface {
	FindByID(ctx context.Context, id uuid.UUID) (*types.Account, error)
	Update(ctx context.Context, account *types.Account) error
}

// OrganizationStore defines the data access interface for organization operations
type OrganizationStore interface {
	FindByID(ctx context.Context, id uuid.UUID) (*types.Organization, error)
}
