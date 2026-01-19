package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Organization struct {
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

func (Organization) TableName() string {
	return "organization"
}
