package profile

import (
	"context"
	"encoding/json"

	"backend/pkg/core"
	"backend/pkg/types"

	"github.com/google/uuid"
)

// Service provides business logic for profile operations
type Service struct {
	profileStore      ProfileStore
	siteSettingsStore SiteSettingsStore
	accountStore      AccountStore
	organizationStore OrganizationStore
}

// NewService creates a new profile service with the provided stores
func NewService(
	profileStore ProfileStore,
	siteSettingsStore SiteSettingsStore,
	accountStore AccountStore,
	organizationStore OrganizationStore,
) *Service {
	return &Service{
		profileStore:      profileStore,
		siteSettingsStore: siteSettingsStore,
		accountStore:      accountStore,
		organizationStore: organizationStore,
	}
}

// GetPublicProfile retrieves the public profile based on site settings
func (s *Service) GetPublicProfile(ctx context.Context) (*PublicProfileResponse, error) {
	pub, err := s.profileStore.GetPublicProfile(ctx)
	if err != nil {
		return nil, err
	}

	socialLinks := make(map[string]string)
	if pub.SocialLinks != nil {
		data, _ := json.Marshal(pub.SocialLinks)
		json.Unmarshal(data, &socialLinks)
	}

	return &PublicProfileResponse{
		Type:            pub.Type,
		Name:            pub.Name,
		Bio:             stringValue(pub.Bio),
		ImageURL:        stringValue(pub.ImageURL),
		EmailPublic:     stringValue(pub.EmailPublic),
		SocialLinks:     socialLinks,
		MetaDescription: stringValue(pub.MetaDescription),
		WebsiteURL:      pub.WebsiteURL,
	}, nil
}

// GetUserProfile returns the profile for a specific user
func (s *Service) GetUserProfile(ctx context.Context, accountID uuid.UUID) (*UserProfile, error) {
	account, err := s.accountStore.FindByID(ctx, accountID)
	if err != nil {
		return nil, err
	}

	socialLinks := parseSocialLinksFromMap(account.SocialLinks)

	return &UserProfile{
		ID:              account.ID,
		Name:            account.Name,
		Bio:             stringValue(account.Bio),
		ProfileImage:    stringValue(account.ProfileImage),
		EmailPublic:     stringValue(account.EmailPublic),
		SocialLinks:     socialLinks,
		MetaDescription: stringValue(account.MetaDescription),
		OrganizationID:  account.OrganizationID,
	}, nil
}

// UpdateUserProfile updates the profile fields for a user
func (s *Service) UpdateUserProfile(ctx context.Context, accountID uuid.UUID, req ProfileUpdateRequest) (*UserProfile, error) {
	account, err := s.accountStore.FindByID(ctx, accountID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.Name != nil {
		account.Name = *req.Name
	}
	if req.Bio != nil {
		account.Bio = req.Bio
	}
	if req.ProfileImage != nil {
		account.ProfileImage = req.ProfileImage
	}
	if req.EmailPublic != nil {
		account.EmailPublic = req.EmailPublic
	}
	if req.MetaDescription != nil {
		account.MetaDescription = req.MetaDescription
	}
	if req.SocialLinks != nil {
		// Convert map[string]string to map[string]interface{}
		socialLinksInterface := make(map[string]interface{})
		for k, v := range *req.SocialLinks {
			socialLinksInterface[k] = v
		}
		account.SocialLinks = socialLinksInterface
	}

	if err := s.accountStore.Update(ctx, account); err != nil {
		return nil, err
	}

	socialLinks := parseSocialLinksFromMap(account.SocialLinks)

	return &UserProfile{
		ID:              account.ID,
		Name:            account.Name,
		Bio:             stringValue(account.Bio),
		ProfileImage:    stringValue(account.ProfileImage),
		EmailPublic:     stringValue(account.EmailPublic),
		SocialLinks:     socialLinks,
		MetaDescription: stringValue(account.MetaDescription),
		OrganizationID:  account.OrganizationID,
	}, nil
}

// GetSiteSettings returns the current site settings
func (s *Service) GetSiteSettings(ctx context.Context) (*SiteSettingsResponse, error) {
	settings, err := s.siteSettingsStore.Get(ctx)
	if err != nil {
		if err == core.ErrNotFound {
			// Return defaults
			return &SiteSettingsResponse{
				PublicProfileType: "user",
			}, nil
		}
		return nil, err
	}

	return &SiteSettingsResponse{
		PublicProfileType:    settings.PublicProfileType,
		PublicUserID:         settings.PublicUserID,
		PublicOrganizationID: settings.PublicOrganizationID,
	}, nil
}

// UpdateSiteSettings updates the site settings
func (s *Service) UpdateSiteSettings(ctx context.Context, req SiteSettingsUpdateRequest) (*SiteSettingsResponse, error) {
	settings, err := s.siteSettingsStore.Get(ctx)
	if err != nil {
		if err == core.ErrNotFound {
			// Create default settings
			settings = &types.SiteSettings{ID: 1, PublicProfileType: "user"}
		} else {
			return nil, err
		}
	}

	if req.PublicProfileType != nil {
		if *req.PublicProfileType != "user" && *req.PublicProfileType != "organization" {
			return nil, core.ErrValidation
		}
		settings.PublicProfileType = *req.PublicProfileType
	}
	if req.PublicUserID != nil {
		// Verify user exists using account store
		_, err := s.accountStore.FindByID(ctx, *req.PublicUserID)
		if err != nil {
			if err == core.ErrNotFound {
				return nil, core.ErrNotFound
			}
			return nil, err
		}
		settings.PublicUserID = req.PublicUserID
	}
	if req.PublicOrganizationID != nil {
		// Verify organization exists using organization store
		_, err := s.organizationStore.FindByID(ctx, *req.PublicOrganizationID)
		if err != nil {
			if err == core.ErrNotFound {
				return nil, core.ErrNotFound
			}
			return nil, err
		}
		settings.PublicOrganizationID = req.PublicOrganizationID
	}

	if err := s.siteSettingsStore.Save(ctx, settings); err != nil {
		return nil, err
	}

	return &SiteSettingsResponse{
		PublicProfileType:    settings.PublicProfileType,
		PublicUserID:         settings.PublicUserID,
		PublicOrganizationID: settings.PublicOrganizationID,
	}, nil
}

// IsUserAdmin checks if a user has admin role
func (s *Service) IsUserAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	return s.profileStore.IsUserAdmin(ctx, userID)
}

// Helper functions

func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func parseSocialLinksFromMap(data map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range data {
		if str, ok := v.(string); ok {
			result[k] = str
		}
	}
	return result
}
