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

// OrganizationRepository provides data access for organizations
type OrganizationRepository struct {
	db *gorm.DB
}

// NewOrganizationRepository creates a new OrganizationRepository
func NewOrganizationRepository(db *gorm.DB) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}

// orgModelToType converts a database model to types
func orgModelToType(m *models.Organization) *types.Organization {
	var socialLinks map[string]interface{}
	if m.SocialLinks != nil {
		_ = json.Unmarshal(m.SocialLinks, &socialLinks)
	}

	return &types.Organization{
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

// orgTypeToModel converts a types type to database model
func orgTypeToModel(o *types.Organization) *models.Organization {
	var socialLinks datatypes.JSON
	if o.SocialLinks != nil {
		data, _ := json.Marshal(o.SocialLinks)
		socialLinks = datatypes.JSON(data)
	}

	return &models.Organization{
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

// FindByID retrieves an organization by its ID
func (r *OrganizationRepository) FindByID(ctx context.Context, id uuid.UUID) (*types.Organization, error) {
	var model models.Organization
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return orgModelToType(&model), nil
}

// FindBySlug retrieves an organization by its slug
func (r *OrganizationRepository) FindBySlug(ctx context.Context, slug string) (*types.Organization, error) {
	var model models.Organization
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return orgModelToType(&model), nil
}

// List retrieves all organizations
func (r *OrganizationRepository) List(ctx context.Context) ([]types.Organization, error) {
	var orgModels []models.Organization
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&orgModels).Error; err != nil {
		return nil, err
	}

	orgs := make([]types.Organization, len(orgModels))
	for i, m := range orgModels {
		orgs[i] = *orgModelToType(&m)
	}
	return orgs, nil
}

// Save creates a new organization
func (r *OrganizationRepository) Save(ctx context.Context, org *types.Organization) error {
	model := orgTypeToModel(org)
	if model.ID == uuid.Nil {
		model.ID = uuid.New()
		org.ID = model.ID
	}
	return r.db.WithContext(ctx).Create(model).Error
}

// Update updates an existing organization
func (r *OrganizationRepository) Update(ctx context.Context, org *types.Organization) error {
	model := orgTypeToModel(org)
	return r.db.WithContext(ctx).Save(model).Error
}

// Delete removes an organization by its ID
func (r *OrganizationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Organization{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.ErrNotFound
	}
	return nil
}
