package models

import (
	"encoding/json"
	"time"

	"backend/pkg/core/auth"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// AccountModel is the GORM model for accounts
type AccountModel struct {
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
	OrganizationID *uuid.UUID         `json:"organization_id" gorm:"type:uuid"`
	Organization   *OrganizationModel `json:"organization,omitempty" gorm:"foreignKey:OrganizationID"`
}

func (AccountModel) TableName() string {
	return "account"
}

// ToCore converts the GORM model to the domain type
func (m *AccountModel) ToCore() *auth.Account {
	var socialLinks map[string]interface{}
	if m.SocialLinks != nil {
		_ = json.Unmarshal(m.SocialLinks, &socialLinks)
	}

	return &auth.Account{
		ID:              m.ID,
		Name:            m.Name,
		Email:           m.Email,
		PasswordHash:    m.PasswordHash,
		Role:            m.Role,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
		Bio:             m.Bio,
		ProfileImage:    m.ProfileImage,
		EmailPublic:     m.EmailPublic,
		SocialLinks:     socialLinks,
		MetaDescription: m.MetaDescription,
		OrganizationID:  m.OrganizationID,
	}
}

// AccountModelFromCore creates a GORM model from the domain type
func AccountModelFromCore(a *auth.Account) *AccountModel {
	var socialLinks datatypes.JSON
	if a.SocialLinks != nil {
		socialLinks, _ = datatypes.NewJSONType(a.SocialLinks).MarshalJSON()
	}

	return &AccountModel{
		ID:              a.ID,
		Name:            a.Name,
		Email:           a.Email,
		PasswordHash:    a.PasswordHash,
		Role:            a.Role,
		CreatedAt:       a.CreatedAt,
		UpdatedAt:       a.UpdatedAt,
		Bio:             a.Bio,
		ProfileImage:    a.ProfileImage,
		EmailPublic:     a.EmailPublic,
		SocialLinks:     socialLinks,
		MetaDescription: a.MetaDescription,
		OrganizationID:  a.OrganizationID,
	}
}
