package models

import (
	"encoding/json"
	"time"

	"backend/pkg/core/organization"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// OrganizationModel is the GORM model for organizations
type OrganizationModel struct {
	ID              uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Name            string         `json:"name" gorm:"not null"`
	Slug            string         `json:"slug" gorm:"uniqueIndex;not null"`
	Bio             *string        `json:"bio"`
	LogoURL         *string        `json:"logo_url"`
	WebsiteURL      *string        `json:"website_url"`
	EmailPublic     *string        `json:"email_public"`
	SocialLinks     datatypes.JSON `json:"social_links" gorm:"type:jsonb;default:'{}'"`
	MetaDescription *string        `json:"meta_description"`
	CreatedAt       time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
}

func (OrganizationModel) TableName() string {
	return "organization"
}

// ToCore converts the GORM model to the domain type
func (m *OrganizationModel) ToCore() *organization.Organization {
	var socialLinks map[string]interface{}
	if m.SocialLinks != nil {
		_ = json.Unmarshal(m.SocialLinks, &socialLinks)
	}

	return &organization.Organization{
		ID:              m.ID,
		Name:            m.Name,
		Slug:            m.Slug,
		Bio:             m.Bio,
		LogoURL:         m.LogoURL,
		WebsiteURL:      m.WebsiteURL,
		EmailPublic:     m.EmailPublic,
		SocialLinks:     socialLinks,
		MetaDescription: m.MetaDescription,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}

// OrganizationModelFromCore creates a GORM model from the domain type
func OrganizationModelFromCore(o *organization.Organization) *OrganizationModel {
	var socialLinks datatypes.JSON
	if o.SocialLinks != nil {
		socialLinks, _ = datatypes.NewJSONType(o.SocialLinks).MarshalJSON()
	}

	return &OrganizationModel{
		ID:              o.ID,
		Name:            o.Name,
		Slug:            o.Slug,
		Bio:             o.Bio,
		LogoURL:         o.LogoURL,
		WebsiteURL:      o.WebsiteURL,
		EmailPublic:     o.EmailPublic,
		SocialLinks:     socialLinks,
		MetaDescription: o.MetaDescription,
		CreatedAt:       o.CreatedAt,
		UpdatedAt:       o.UpdatedAt,
	}
}
