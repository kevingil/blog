// Package profile provides profile and site settings domain types
package profile

import (
	"backend/pkg/api/dto"
	"backend/pkg/types"
)

// SiteSettings is an alias to types.SiteSettings for backward compatibility
type SiteSettings = types.SiteSettings

// PublicProfile is an alias to types.PublicProfile for backward compatibility
type PublicProfile = types.PublicProfile

// UserProfile is an alias to types.UserProfile for backward compatibility
type UserProfile = types.UserProfile

// ProfileUpdateRequest is an alias to dto.ProfileUpdateRequest for backward compatibility
type ProfileUpdateRequest = dto.ProfileUpdateRequest

// PublicProfileResponse is an alias to dto.PublicProfileResponse for backward compatibility
type PublicProfileResponse = dto.PublicProfileResponse

// SiteSettingsResponse is an alias to dto.SiteSettingsResponse for backward compatibility
type SiteSettingsResponse = dto.SiteSettingsResponse

// SiteSettingsUpdateRequest is an alias to dto.SiteSettingsUpdateRequest for backward compatibility
type SiteSettingsUpdateRequest = dto.SiteSettingsUpdateRequest

// UserProfileResponse is an alias to dto.UserProfileResponse for backward compatibility
type UserProfileResponse = dto.UserProfileResponse
