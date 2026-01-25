package repository

import (
	"context"
	"encoding/json"

	"backend/pkg/core"
	"backend/pkg/database/models"
	"backend/pkg/types"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SiteSettingsRepository provides data access for site settings
type SiteSettingsRepository struct {
	db *gorm.DB
}

// NewSiteSettingsRepository creates a new SiteSettingsRepository
func NewSiteSettingsRepository(db *gorm.DB) *SiteSettingsRepository {
	return &SiteSettingsRepository{db: db}
}

// siteSettingsModelToType converts a database model to types
func siteSettingsModelToType(m *models.SiteSettings) *types.SiteSettings {
	return &types.SiteSettings{
		ID:                   m.ID,
		PublicProfileType:    m.PublicProfileType,
		PublicUserID:         m.PublicUserID,
		PublicOrganizationID: m.PublicOrganizationID,
		CreatedAt:            m.CreatedAt,
		UpdatedAt:            m.UpdatedAt,
	}
}

// siteSettingsTypeToModel converts a types type to database model
func siteSettingsTypeToModel(s *types.SiteSettings) *models.SiteSettings {
	return &models.SiteSettings{
		ID:                   s.ID,
		PublicProfileType:    s.PublicProfileType,
		PublicUserID:         s.PublicUserID,
		PublicOrganizationID: s.PublicOrganizationID,
		CreatedAt:            s.CreatedAt,
		UpdatedAt:            s.UpdatedAt,
	}
}

// Get retrieves the site settings (there's only one row)
func (r *SiteSettingsRepository) Get(ctx context.Context) (*types.SiteSettings, error) {
	var model models.SiteSettings
	if err := r.db.WithContext(ctx).First(&model, 1).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return siteSettingsModelToType(&model), nil
}

// Save creates or updates site settings
func (r *SiteSettingsRepository) Save(ctx context.Context, settings *types.SiteSettings) error {
	model := siteSettingsTypeToModel(settings)
	model.ID = 1 // Always use ID 1 for site settings
	settings.ID = 1
	return r.db.WithContext(ctx).Save(model).Error
}

// ProfileRepository provides data access for profile-related operations
type ProfileRepository struct {
	db *gorm.DB
}

// NewProfileRepository creates a new ProfileRepository
func NewProfileRepository(db *gorm.DB) *ProfileRepository {
	return &ProfileRepository{db: db}
}

// GetPublicProfile retrieves the public profile based on site settings
func (r *ProfileRepository) GetPublicProfile(ctx context.Context) (*types.PublicProfile, error) {
	var settings models.SiteSettings
	if err := r.db.WithContext(ctx).
		Preload("PublicUser").
		Preload("PublicOrganization").
		First(&settings, 1).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}

	pub := &types.PublicProfile{
		Type: settings.PublicProfileType,
	}

	if settings.PublicProfileType == "organization" && settings.PublicOrganization != nil {
		org := settings.PublicOrganization
		var socialLinks map[string]interface{}
		if org.SocialLinks != nil {
			_ = json.Unmarshal(org.SocialLinks, &socialLinks)
		}
		pub.Name = org.Name
		pub.Bio = org.Bio
		pub.ImageURL = org.LogoURL
		pub.WebsiteURL = org.WebsiteURL
		pub.EmailPublic = org.EmailPublic
		pub.SocialLinks = socialLinks
		pub.MetaDescription = org.MetaDescription
	} else if settings.PublicUser != nil {
		user := settings.PublicUser
		var socialLinks map[string]interface{}
		if user.SocialLinks != nil {
			_ = json.Unmarshal(user.SocialLinks, &socialLinks)
		}
		pub.Name = user.Name
		pub.Bio = user.Bio
		pub.ImageURL = user.ProfileImage
		pub.EmailPublic = user.EmailPublic
		pub.SocialLinks = socialLinks
		pub.MetaDescription = user.MetaDescription
	}

	return pub, nil
}

// IsUserAdmin checks if a user has admin role
func (r *ProfileRepository) IsUserAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	var model models.Account
	if err := r.db.WithContext(ctx).Select("role").First(&model, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, core.ErrNotFound
		}
		return false, err
	}
	return model.Role == "admin", nil
}
