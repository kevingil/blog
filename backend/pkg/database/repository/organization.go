package repository

import (
	"context"

	"backend/pkg/core"
	"backend/pkg/core/organization"
	"backend/pkg/database/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OrganizationRepository implements organization.OrganizationStore using GORM
type OrganizationRepository struct {
	db *gorm.DB
}

// NewOrganizationRepository creates a new OrganizationRepository
func NewOrganizationRepository(db *gorm.DB) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}

// FindByID retrieves an organization by its ID
func (r *OrganizationRepository) FindByID(ctx context.Context, id uuid.UUID) (*organization.Organization, error) {
	var model models.Organization
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return model.ToCore(), nil
}

// FindBySlug retrieves an organization by its slug
func (r *OrganizationRepository) FindBySlug(ctx context.Context, slug string) (*organization.Organization, error) {
	var model models.Organization
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return model.ToCore(), nil
}

// List retrieves all organizations
func (r *OrganizationRepository) List(ctx context.Context) ([]organization.Organization, error) {
	var orgModels []models.Organization
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&orgModels).Error; err != nil {
		return nil, err
	}

	orgs := make([]organization.Organization, len(orgModels))
	for i, m := range orgModels {
		orgs[i] = *m.ToCore()
	}
	return orgs, nil
}

// Save creates a new organization
func (r *OrganizationRepository) Save(ctx context.Context, o *organization.Organization) error {
	model := models.OrganizationFromCore(o)

	if o.ID == uuid.Nil {
		o.ID = uuid.New()
		model.ID = o.ID
	}

	return r.db.WithContext(ctx).Create(model).Error
}

// Update updates an existing organization
func (r *OrganizationRepository) Update(ctx context.Context, o *organization.Organization) error {
	model := models.OrganizationFromCore(o)
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
