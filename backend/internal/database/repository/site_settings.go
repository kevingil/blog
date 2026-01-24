package repository

import (
	"context"

	"blog-agent-go/backend/internal/core"
	"blog-agent-go/backend/internal/core/profile"
	"blog-agent-go/backend/internal/database/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SiteSettingsRepository implements profile.SiteSettingsStore using GORM
type SiteSettingsRepository struct {
	db *gorm.DB
}

// NewSiteSettingsRepository creates a new SiteSettingsRepository
func NewSiteSettingsRepository(db *gorm.DB) *SiteSettingsRepository {
	return &SiteSettingsRepository{db: db}
}

// Get retrieves the site settings (there's only one row)
func (r *SiteSettingsRepository) Get(ctx context.Context) (*profile.SiteSettings, error) {
	var model models.SiteSettingsModel
	if err := r.db.WithContext(ctx).First(&model, 1).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return model.ToCore(), nil
}

// Save creates or updates site settings
func (r *SiteSettingsRepository) Save(ctx context.Context, s *profile.SiteSettings) error {
	model := models.SiteSettingsModelFromCore(s)
	model.ID = 1 // Always use ID 1 for site settings

	return r.db.WithContext(ctx).Save(model).Error
}

// ProfileRepository implements profile.ProfileStore using GORM
type ProfileRepository struct {
	db *gorm.DB
}

// NewProfileRepository creates a new ProfileRepository
func NewProfileRepository(db *gorm.DB) *ProfileRepository {
	return &ProfileRepository{db: db}
}

// GetPublicProfile retrieves the public profile based on site settings
func (r *ProfileRepository) GetPublicProfile(ctx context.Context) (*profile.PublicProfile, error) {
	var settings models.SiteSettingsModel
	if err := r.db.WithContext(ctx).
		Preload("PublicUser").
		Preload("PublicOrganization").
		First(&settings, 1).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}

	pub := &profile.PublicProfile{
		Type: settings.PublicProfileType,
	}

	if settings.PublicProfileType == "organization" && settings.PublicOrganization != nil {
		org := settings.PublicOrganization
		var socialLinks map[string]interface{}
		if org.SocialLinks != nil {
			_ = org.SocialLinks.Unmarshal(&socialLinks)
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
			_ = user.SocialLinks.Unmarshal(&socialLinks)
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
	var model models.AccountModel
	if err := r.db.WithContext(ctx).Select("role").First(&model, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, core.ErrNotFound
		}
		return false, err
	}
	return model.Role == "admin", nil
}
