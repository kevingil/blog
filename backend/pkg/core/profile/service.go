package profile

import (
	"context"

	"github.com/google/uuid"
)

// Service provides business logic for profiles and site settings
type Service struct {
	settingsStore SiteSettingsStore
	profileStore  ProfileStore
}

// NewService creates a new profile service
func NewService(settingsStore SiteSettingsStore, profileStore ProfileStore) *Service {
	return &Service{
		settingsStore: settingsStore,
		profileStore:  profileStore,
	}
}

// GetSiteSettings retrieves the site settings
func (s *Service) GetSiteSettings(ctx context.Context) (*SiteSettings, error) {
	return s.settingsStore.Get(ctx)
}

// UpdateSiteSettings updates the site settings
func (s *Service) UpdateSiteSettings(ctx context.Context, settings *SiteSettings) error {
	return s.settingsStore.Save(ctx, settings)
}

// GetPublicProfile retrieves the public profile based on site settings
func (s *Service) GetPublicProfile(ctx context.Context) (*PublicProfile, error) {
	return s.profileStore.GetPublicProfile(ctx)
}

// IsUserAdmin checks if a user has admin privileges
func (s *Service) IsUserAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	return s.profileStore.IsUserAdmin(ctx, userID)
}

// UpdateSiteSettingsRequest represents a request to update site settings
type UpdateSiteSettingsRequest struct {
	PublicProfileType    string
	PublicUserID         *uuid.UUID
	PublicOrganizationID *uuid.UUID
}

// UpdateSettings updates site settings with validation
func (s *Service) UpdateSettings(ctx context.Context, req UpdateSiteSettingsRequest) error {
	settings, err := s.settingsStore.Get(ctx)
	if err != nil {
		// Create new settings if not found
		settings = &SiteSettings{
			ID: 1,
		}
	}

	settings.PublicProfileType = req.PublicProfileType
	settings.PublicUserID = req.PublicUserID
	settings.PublicOrganizationID = req.PublicOrganizationID

	return s.settingsStore.Save(ctx, settings)
}
