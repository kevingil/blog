package services

import (
	"backend/pkg/core"
	"backend/pkg/database"
	"backend/pkg/database/models"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ProfileService provides methods to interact with user profiles and site settings
type ProfileService struct {
	db database.Service
}

var profileSvc *ProfileService

// InitProfileService initializes the profile service singleton
func InitProfileService() {
	if profileSvc == nil {
		profileSvc = NewProfileService(database.New())
	}
}

// Profiles returns the profile service singleton
func Profiles() *ProfileService {
	if profileSvc == nil {
		log.Fatal("ProfileService not initialized. Call InitProfileService() first.")
	}
	return profileSvc
}

func NewProfileService(db database.Service) *ProfileService {
	return &ProfileService{db: db}
}

// PublicProfile is a unified response for both user and organization profiles
type PublicProfile struct {
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

// GetPublicProfile returns the public profile based on site settings
func (s *ProfileService) GetPublicProfile() (*PublicProfile, error) {
	db := s.db.GetDB()

	// Get site settings
	var settings models.SiteSettings
	if err := db.First(&settings, "id = 1").Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, core.InternalError("Failed to fetch site settings")
	}

	if settings.PublicProfileType == "organization" && settings.PublicOrganizationID != nil {
		// Return organization profile
		var org models.Organization
		if err := db.First(&org, "id = ?", settings.PublicOrganizationID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, nil
			}
			return nil, core.InternalError("Failed to fetch organization")
		}

		socialLinks := parseSocialLinks(org.SocialLinks)

		return &PublicProfile{
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
		fmt.Println("No public user ID set")
		return nil, nil
	}

	var account models.Account
	if err := db.First(&account, "id = ?", settings.PublicUserID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, core.InternalError("Failed to fetch account")
	}

	socialLinks := parseSocialLinks(account.SocialLinks)

	return &PublicProfile{
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
func (s *ProfileService) GetUserProfile(accountID uuid.UUID) (*UserProfile, error) {
	db := s.db.GetDB()
	fmt.Println("Getting user profile for account ID:", accountID)
	var account models.Account
	if err := db.First(&account, "id = ?", accountID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.NotFoundError("Account")
		}
		return nil, core.InternalError("Failed to fetch account")
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
func (s *ProfileService) UpdateUserProfile(accountID uuid.UUID, req ProfileUpdateRequest) (*UserProfile, error) {
	db := s.db.GetDB()

	var account models.Account
	if err := db.First(&account, "id = ?", accountID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.NotFoundError("Account")
		}
		return nil, core.InternalError("Failed to fetch account")
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
			return nil, core.InternalError("Failed to marshal social_links")
		}
		updates["social_links"] = datatypes.JSON(socialLinksJSON)
	}

	if len(updates) > 0 {
		if err := db.Model(&account).Updates(updates).Error; err != nil {
			return nil, core.InternalError("Failed to update profile")
		}
	}

	// Reload the account
	if err := db.First(&account, "id = ?", accountID).Error; err != nil {
		return nil, core.InternalError("Failed to reload account")
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
func (s *ProfileService) GetSiteSettings() (*SiteSettingsResponse, error) {
	db := s.db.GetDB()

	var settings models.SiteSettings
	if err := db.First(&settings, "id = 1").Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Return defaults
			return &SiteSettingsResponse{
				PublicProfileType: "user",
			}, nil
		}
		return nil, core.InternalError("Failed to fetch site settings")
	}

	return &SiteSettingsResponse{
		PublicProfileType:    settings.PublicProfileType,
		PublicUserID:         settings.PublicUserID,
		PublicOrganizationID: settings.PublicOrganizationID,
	}, nil
}

// UpdateSiteSettings updates the site settings
func (s *ProfileService) UpdateSiteSettings(req SiteSettingsUpdateRequest) (*SiteSettingsResponse, error) {
	db := s.db.GetDB()

	var settings models.SiteSettings
	if err := db.First(&settings, "id = 1").Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create default settings
			settings = models.SiteSettings{ID: 1, PublicProfileType: "user"}
			if err := db.Create(&settings).Error; err != nil {
				return nil, core.InternalError("Failed to create site settings")
			}
		} else {
			return nil, core.InternalError("Failed to fetch site settings")
		}
	}

	updates := make(map[string]interface{})

	if req.PublicProfileType != nil {
		if *req.PublicProfileType != "user" && *req.PublicProfileType != "organization" {
			return nil, core.InvalidInputError("public_profile_type must be 'user' or 'organization'")
		}
		updates["public_profile_type"] = *req.PublicProfileType
	}
	if req.PublicUserID != nil {
		// Verify user exists
		var count int64
		db.Model(&models.Account{}).Where("id = ?", req.PublicUserID).Count(&count)
		if count == 0 {
			return nil, core.NotFoundError("User")
		}
		updates["public_user_id"] = *req.PublicUserID
	}
	if req.PublicOrganizationID != nil {
		// Verify organization exists
		var count int64
		db.Model(&models.Organization{}).Where("id = ?", req.PublicOrganizationID).Count(&count)
		if count == 0 {
			return nil, core.NotFoundError("Organization")
		}
		updates["public_organization_id"] = *req.PublicOrganizationID
	}

	if len(updates) > 0 {
		if err := db.Model(&settings).Updates(updates).Error; err != nil {
			return nil, core.InternalError("Failed to update site settings")
		}
	}

	// Reload settings
	if err := db.First(&settings, "id = 1").Error; err != nil {
		return nil, core.InternalError("Failed to reload site settings")
	}

	return &SiteSettingsResponse{
		PublicProfileType:    settings.PublicProfileType,
		PublicUserID:         settings.PublicUserID,
		PublicOrganizationID: settings.PublicOrganizationID,
	}, nil
}

// IsUserAdmin checks if a user has admin role
func (s *ProfileService) IsUserAdmin(userID uuid.UUID) (bool, error) {
	db := s.db.GetDB()

	var account models.Account
	if err := db.Select("role").First(&account, "id = ?", userID).Error; err != nil {
		return false, core.InternalError("Failed to fetch account")
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
