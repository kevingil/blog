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

// PageRepository provides data access for pages
type PageRepository struct {
	db *gorm.DB
}

// NewPageRepository creates a new PageRepository
func NewPageRepository(db *gorm.DB) *PageRepository {
	return &PageRepository{db: db}
}

// pageModelToType converts a database model to types
func pageModelToType(m *models.Page) *types.Page {
	var metaData map[string]interface{}
	if m.MetaData != nil {
		_ = json.Unmarshal(m.MetaData, &metaData)
	}

	return &types.Page{
		ID:          m.ID,
		Slug:        m.Slug,
		Title:       m.Title,
		Content:     m.Content,
		Description: m.Description,
		ImageURL:    m.ImageURL,
		MetaData:    metaData,
		IsPublished: m.IsPublished,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// pageTypeToModel converts a types type to database model
func pageTypeToModel(p *types.Page) *models.Page {
	var metaData datatypes.JSON
	if p.MetaData != nil {
		data, _ := json.Marshal(p.MetaData)
		metaData = datatypes.JSON(data)
	}

	return &models.Page{
		ID:          p.ID,
		Slug:        p.Slug,
		Title:       p.Title,
		Content:     p.Content,
		Description: p.Description,
		ImageURL:    p.ImageURL,
		MetaData:    metaData,
		IsPublished: p.IsPublished,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// FindByID retrieves a page by its ID
func (r *PageRepository) FindByID(ctx context.Context, id uuid.UUID) (*types.Page, error) {
	var model models.Page
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return pageModelToType(&model), nil
}

// FindBySlug retrieves a page by its slug
func (r *PageRepository) FindBySlug(ctx context.Context, slug string) (*types.Page, error) {
	var model models.Page
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return pageModelToType(&model), nil
}

// List retrieves pages with pagination
func (r *PageRepository) List(ctx context.Context, opts types.PageListOptions) ([]types.Page, int64, error) {
	var pageModels []models.Page
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Page{})

	// Apply IsPublished filter if specified
	if opts.IsPublished != nil {
		query = query.Where("is_published = ?", *opts.IsPublished)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	offset := (opts.Page - 1) * opts.PerPage
	if err := query.Offset(offset).Limit(opts.PerPage).Order("created_at DESC").Find(&pageModels).Error; err != nil {
		return nil, 0, err
	}

	pages := make([]types.Page, len(pageModels))
	for i, m := range pageModels {
		pages[i] = *pageModelToType(&m)
	}

	return pages, total, nil
}

// Save creates or updates a page
func (r *PageRepository) Save(ctx context.Context, page *types.Page) error {
	model := pageTypeToModel(page)

	// Check if page exists
	var existing models.Page
	err := r.db.WithContext(ctx).First(&existing, model.ID).Error
	if err == gorm.ErrRecordNotFound {
		// Create new page
		if model.ID == uuid.Nil {
			model.ID = uuid.New()
			page.ID = model.ID
		}
		return r.db.WithContext(ctx).Create(model).Error
	} else if err != nil {
		return err
	}

	// Update existing page
	return r.db.WithContext(ctx).Save(model).Error
}

// Delete removes a page by its ID
func (r *PageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Page{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.ErrNotFound
	}
	return nil
}
