package models

import (
	"time"

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
