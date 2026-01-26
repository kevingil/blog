package repository

import (
	"context"
	"encoding/json"

	"backend/pkg/core"
	"backend/pkg/database/models"
	"backend/pkg/types"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// AccountRepository provides data access for accounts
type AccountRepository struct {
	db *gorm.DB
}

// NewAccountRepository creates a new AccountRepository
func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

// accountModelToSchema converts a database model to types
func accountModelToSchema(m *models.Account) *types.Account {
	var socialLinks map[string]interface{}
	if m.SocialLinks != nil {
		_ = json.Unmarshal(m.SocialLinks, &socialLinks)
	}

	return &types.Account{
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

// accountSchemaToModel converts a types type to database model
func accountSchemaToModel(s *types.Account) *models.Account {
	var socialLinks datatypes.JSON
	if s.SocialLinks != nil {
		data, _ := json.Marshal(s.SocialLinks)
		socialLinks = datatypes.JSON(data)
	}

	return &models.Account{
		ID:              s.ID,
		Name:            s.Name,
		Email:           s.Email,
		PasswordHash:    s.PasswordHash,
		Role:            s.Role,
		CreatedAt:       s.CreatedAt,
		UpdatedAt:       s.UpdatedAt,
		Bio:             s.Bio,
		ProfileImage:    s.ProfileImage,
		EmailPublic:     s.EmailPublic,
		SocialLinks:     socialLinks,
		MetaDescription: s.MetaDescription,
		OrganizationID:  s.OrganizationID,
	}
}

// FindByID retrieves an account by its ID
func (r *AccountRepository) FindByID(ctx context.Context, id uuid.UUID) (*types.Account, error) {
	var model models.Account
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return accountModelToSchema(&model), nil
}

// FindByEmail retrieves an account by its email
func (r *AccountRepository) FindByEmail(ctx context.Context, email string) (*types.Account, error) {
	var model models.Account
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return accountModelToSchema(&model), nil
}

// Save creates a new account
func (r *AccountRepository) Save(ctx context.Context, account *types.Account) error {
	model := accountSchemaToModel(account)
	if model.ID == uuid.Nil {
		model.ID = uuid.New()
		account.ID = model.ID
	}
	return r.db.WithContext(ctx).Create(model).Error
}

// Update updates an existing account
func (r *AccountRepository) Update(ctx context.Context, account *types.Account) error {
	model := accountSchemaToModel(account)
	return r.db.WithContext(ctx).Save(model).Error
}

// Delete removes an account by its ID
func (r *AccountRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Account{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.ErrNotFound
	}
	return nil
}
