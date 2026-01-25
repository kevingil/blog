package models

import (
	"time"

	"backend/pkg/core/profile"

	"github.com/google/uuid"
)

// SiteSettings is the GORM model for site settings
type SiteSettings struct {
	ID                   int        `json:"id" gorm:"primaryKey;default:1"`
	PublicProfileType    string     `json:"public_profile_type" gorm:"default:user"` // 'user' or 'organization'
	PublicUserID         *uuid.UUID `json:"public_user_id" gorm:"type:uuid"`
	PublicOrganizationID *uuid.UUID `json:"public_organization_id" gorm:"type:uuid"`
	CreatedAt            time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt            time.Time  `json:"updated_at" gorm:"autoUpdateTime"`

	// Relationships for eager loading
	PublicUser         *Account      `json:"public_user,omitempty" gorm:"foreignKey:PublicUserID"`
	PublicOrganization *Organization `json:"public_organization,omitempty" gorm:"foreignKey:PublicOrganizationID"`
}

func (SiteSettings) TableName() string {
	return "site_settings"
}

// ToCore converts the GORM model to the domain type
func (m *SiteSettings) ToCore() *profile.SiteSettings {
	return &profile.SiteSettings{
		ID:                   m.ID,
		PublicProfileType:    m.PublicProfileType,
		PublicUserID:         m.PublicUserID,
		PublicOrganizationID: m.PublicOrganizationID,
		CreatedAt:            m.CreatedAt,
		UpdatedAt:            m.UpdatedAt,
	}
}

// SiteSettingsFromCore creates a GORM model from the domain type
func SiteSettingsFromCore(s *profile.SiteSettings) *SiteSettings {
	return &SiteSettings{
		ID:                   s.ID,
		PublicProfileType:    s.PublicProfileType,
		PublicUserID:         s.PublicUserID,
		PublicOrganizationID: s.PublicOrganizationID,
		CreatedAt:            s.CreatedAt,
		UpdatedAt:            s.UpdatedAt,
	}
}
