package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Account is the GORM model for accounts
type Account struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Name         string    `json:"name" gorm:"not null"`
	Email        string    `json:"email" gorm:"uniqueIndex;not null"`
	PasswordHash string    `json:"-" gorm:"not null;column:password_hash"`
	Role         string    `json:"role" gorm:"default:user;not null"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// Profile fields
	Bio             *string        `json:"bio"`
	ProfileImage    *string        `json:"profile_image"`
	EmailPublic     *string        `json:"email_public"`
	SocialLinks     datatypes.JSON `json:"social_links" gorm:"type:jsonb;default:'{}'"`
	MetaDescription *string        `json:"meta_description"`

	// Organization relationship (optional)
	OrganizationID *uuid.UUID    `json:"organization_id" gorm:"type:uuid"`
	Organization   *Organization `json:"organization,omitempty" gorm:"foreignKey:OrganizationID"`
}

func (Account) TableName() string {
	return "account"
}
