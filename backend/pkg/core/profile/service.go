package profile

import (
	"context"
	"encoding/json"

	"backend/pkg/core"
	"backend/pkg/database"
	"backend/pkg/database/models"
	"backend/pkg/database/repository"
	"backend/pkg/types"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// getSettingsRepo returns a site settings repository instance
func getSettingsRepo() *repository.SiteSettingsRepository {
	return repository.NewSiteSettingsRepository(database.DB())
}

// getProfileRepo returns a profile repository instance
func getProfileRepo() *repository.ProfileRepository {
	return repository.NewProfileRepository(database.DB())
}

// GetPublicProfile retrieves the public profile based on site settings
func GetPublicProfile(ctx context.Context) (*PublicProfileResponse, error) {
	repo := getProfileRepo()
	pub, err := repo.GetPublicProfile(ctx)
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
	repo := getSettingsRepo()

	settings, err := repo.Get(ctx)
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
func UpdateSiteSettings(ctx context.Context, req SiteSettingsUpdateRequest) (*SiteSettingsResponse, error) {
	db := database.DB()
	repo := getSettingsRepo()

	settings, err := repo.Get(ctx)
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
		// Verify user exists
		var count int64
		db.WithContext(ctx).Model(&models.Account{}).Where("id = ?", req.PublicUserID).Count(&count)
		if count == 0 {
			return nil, core.ErrNotFound
		}
		settings.PublicUserID = req.PublicUserID
	}
	if req.PublicOrganizationID != nil {
		// Verify organization exists
		var count int64
		db.WithContext(ctx).Model(&models.Organization{}).Where("id = ?", req.PublicOrganizationID).Count(&count)
		if count == 0 {
			return nil, core.ErrNotFound
		}
		settings.PublicOrganizationID = req.PublicOrganizationID
	}

	if err := repo.Save(ctx, settings); err != nil {
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
	repo := getProfileRepo()
	return repo.IsUserAdmin(ctx, userID)
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

