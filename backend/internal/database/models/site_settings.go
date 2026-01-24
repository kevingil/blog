package models

import (
	"time"

	"blog-agent-go/backend/internal/core/profile"

	"github.com/google/uuid"
)

// SiteSettingsModel is the GORM model for site settings
type SiteSettingsModel struct {
	ID                   int        `json:"id" gorm:"primaryKey;default:1"`
	PublicProfileType    string     `json:"public_profile_type" gorm:"default:user"` // 'user' or 'organization'
	PublicUserID         *uuid.UUID `json:"public_user_id" gorm:"type:uuid"`
	PublicOrganizationID *uuid.UUID `json:"public_organization_id" gorm:"type:uuid"`
	CreatedAt            time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt            time.Time  `json:"updated_at" gorm:"autoUpdateTime"`

	// Relationships for eager loading
	PublicUser         *AccountModel      `json:"public_user,omitempty" gorm:"foreignKey:PublicUserID"`
	PublicOrganization *OrganizationModel `json:"public_organization,omitempty" gorm:"foreignKey:PublicOrganizationID"`
}

func (SiteSettingsModel) TableName() string {
	return "site_settings"
}

// ToCore converts the GORM model to the domain type
func (m *SiteSettingsModel) ToCore() *profile.SiteSettings {
	return &profile.SiteSettings{
		ID:                   m.ID,
		PublicProfileType:    m.PublicProfileType,
		PublicUserID:         m.PublicUserID,
		PublicOrganizationID: m.PublicOrganizationID,
		CreatedAt:            m.CreatedAt,
		UpdatedAt:            m.UpdatedAt,
	}
}

// SiteSettingsModelFromCore creates a GORM model from the domain type
func SiteSettingsModelFromCore(s *profile.SiteSettings) *SiteSettingsModel {
	return &SiteSettingsModel{
		ID:                   s.ID,
		PublicProfileType:    s.PublicProfileType,
		PublicUserID:         s.PublicUserID,
		PublicOrganizationID: s.PublicOrganizationID,
		CreatedAt:            s.CreatedAt,
		UpdatedAt:            s.UpdatedAt,
	}
}
