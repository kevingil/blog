package profile

import (
	"context"
	"encoding/json"

	"backend/pkg/core"
	"backend/pkg/database"
	"backend/pkg/database/models"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// UserProfile is the profile data for a user account
type UserProfile struct {
	ID              uuid.UUID         `json:"id"`
	Name            string            `json:"name"`
	Bio             string            `json:"bio"`
	ProfileImage    string            `json:"profile_image"`
	EmailPublic     string            `json:"email_public"`
	SocialLinks     map[string]string `json:"social_links"`
	MetaDescription string            `json:"meta_description"`
	OrganizationID  *uuid.UUID        `json:"organization_id,omitempty"`
}

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

// GetPublicProfile retrieves the public profile based on site settings
func GetPublicProfile(ctx context.Context) (*PublicProfileResponse, error) {
	db := database.DB()

	// Get site settings
	var settings models.SiteSettings
	if err := db.WithContext(ctx).First(&settings, "id = 1").Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}

	if settings.PublicProfileType == "organization" && settings.PublicOrganizationID != nil {
		// Return organization profile
		var org models.Organization
		if err := db.WithContext(ctx).First(&org, "id = ?", settings.PublicOrganizationID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, core.ErrNotFound
			}
			return nil, err
		}

		socialLinks := parseSocialLinks(org.SocialLinks)

		return &PublicProfileResponse{
			Type:            "organization",
			ID:              org.ID,
			Name:            org.Name,
			Bio:             stringValue(org.Bio),
			ImageURL:        stringValue(org.LogoURL),
			EmailPublic:     stringValue(org.EmailPublic),
			SocialLinks:     socialLinks,
			MetaDescription: stringValue(org.MetaDescription),
			WebsiteURL:      org.WebsiteURL,
		}, nil
	}

	// Return user profile (default)
	if settings.PublicUserID == nil {
		return nil, core.ErrNotFound
	}

	var account models.Account
	if err := db.WithContext(ctx).First(&account, "id = ?", settings.PublicUserID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}

	socialLinks := parseSocialLinks(account.SocialLinks)

	return &PublicProfileResponse{
		Type:            "user",
		ID:              account.ID,
		Name:            account.Name,
		Bio:             stringValue(account.Bio),
		ImageURL:        stringValue(account.ProfileImage),
		EmailPublic:     stringValue(account.EmailPublic),
		SocialLinks:     socialLinks,
		MetaDescription: stringValue(account.MetaDescription),
	}, nil
}

// GetUserProfile returns the profile for a specific user
func GetUserProfile(ctx context.Context, accountID uuid.UUID) (*UserProfile, error) {
	db := database.DB()

	var account models.Account
	if err := db.WithContext(ctx).First(&account, "id = ?", accountID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}

	socialLinks := parseSocialLinks(account.SocialLinks)

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
func UpdateUserProfile(ctx context.Context, accountID uuid.UUID, req ProfileUpdateRequest) (*UserProfile, error) {
	db := database.DB()

	var account models.Account
	if err := db.WithContext(ctx).First(&account, "id = ?", accountID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}

	updates := make(map[string]interface{})

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Bio != nil {
		updates["bio"] = *req.Bio
	}
	if req.ProfileImage != nil {
		updates["profile_image"] = *req.ProfileImage
	}
	if req.EmailPublic != nil {
		updates["email_public"] = *req.EmailPublic
	}
	if req.MetaDescription != nil {
		updates["meta_description"] = *req.MetaDescription
	}
	if req.SocialLinks != nil {
		socialLinksJSON, err := json.Marshal(*req.SocialLinks)
		if err != nil {
			return nil, err
		}
		updates["social_links"] = datatypes.JSON(socialLinksJSON)
	}

	if len(updates) > 0 {
		if err := db.WithContext(ctx).Model(&account).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	// Reload the account
	if err := db.WithContext(ctx).First(&account, "id = ?", accountID).Error; err != nil {
		return nil, err
	}

	socialLinks := parseSocialLinks(account.SocialLinks)

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
func GetSiteSettings(ctx context.Context) (*SiteSettingsResponse, error) {
	db := database.DB()

	var settings models.SiteSettings
	if err := db.WithContext(ctx).First(&settings, "id = 1").Error; err != nil {
		if err == gorm.ErrRecordNotFound {
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
func UpdateSiteSettings(ctx context.Context, req SiteSettingsUpdateRequest) (*SiteSettingsResponse, error) {
	db := database.DB()

	var settings models.SiteSettings
	if err := db.WithContext(ctx).First(&settings, "id = 1").Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create default settings
			settings = models.SiteSettings{ID: 1, PublicProfileType: "user"}
			if err := db.WithContext(ctx).Create(&settings).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	updates := make(map[string]interface{})

	if req.PublicProfileType != nil {
		if *req.PublicProfileType != "user" && *req.PublicProfileType != "organization" {
			return nil, core.ErrValidation
		}
		updates["public_profile_type"] = *req.PublicProfileType
	}
	if req.PublicUserID != nil {
		// Verify user exists
		var count int64
		db.WithContext(ctx).Model(&models.Account{}).Where("id = ?", req.PublicUserID).Count(&count)
		if count == 0 {
			return nil, core.ErrNotFound
		}
		updates["public_user_id"] = *req.PublicUserID
	}
	if req.PublicOrganizationID != nil {
		// Verify organization exists
		var count int64
		db.WithContext(ctx).Model(&models.Organization{}).Where("id = ?", req.PublicOrganizationID).Count(&count)
		if count == 0 {
			return nil, core.ErrNotFound
		}
		updates["public_organization_id"] = *req.PublicOrganizationID
	}

	if len(updates) > 0 {
		if err := db.WithContext(ctx).Model(&settings).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	// Reload settings
	if err := db.WithContext(ctx).First(&settings, "id = 1").Error; err != nil {
		return nil, err
	}

	return &SiteSettingsResponse{
		PublicProfileType:    settings.PublicProfileType,
		PublicUserID:         settings.PublicUserID,
		PublicOrganizationID: settings.PublicOrganizationID,
	}, nil
}

// IsUserAdmin checks if a user has admin role
func IsUserAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	db := database.DB()

	var account models.Account
	if err := db.WithContext(ctx).Select("role").First(&account, "id = ?", userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, core.ErrNotFound
		}
		return false, err
	}

	return account.Role == "admin", nil
}

// Helper functions

func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func parseSocialLinks(data datatypes.JSON) map[string]string {
	result := make(map[string]string)
	if data != nil {
		_ = json.Unmarshal(data, &result)
	}
	return result
}

// Legacy Service type for backward compatibility during migration
// TODO: Remove after full migration to package-level functions

// Service provides business logic for profiles and site settings (deprecated - use package functions)
type Service struct {
	settingsStore SiteSettingsStore
	profileStore  ProfileStore
}

// NewService creates a new profile service (deprecated - use package functions)
func NewService(settingsStore SiteSettingsStore, profileStore ProfileStore) *Service {
	return &Service{
		settingsStore: settingsStore,
		profileStore:  profileStore,
	}
}
