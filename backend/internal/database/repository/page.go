package repository

import (
	"context"

	"blog-agent-go/backend/internal/core"
	"blog-agent-go/backend/internal/core/page"
	"blog-agent-go/backend/internal/database/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PageRepository implements page.PageStore using GORM
type PageRepository struct {
	db *gorm.DB
}

// NewPageRepository creates a new PageRepository
func NewPageRepository(db *gorm.DB) *PageRepository {
	return &PageRepository{db: db}
}

// FindByID retrieves a page by its ID
func (r *PageRepository) FindByID(ctx context.Context, id uuid.UUID) (*page.Page, error) {
	var model models.PageModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return model.ToCore(), nil
}

// FindBySlug retrieves a page by its slug
func (r *PageRepository) FindBySlug(ctx context.Context, slug string) (*page.Page, error) {
	var model models.PageModel
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return model.ToCore(), nil
}

// List retrieves pages with pagination
func (r *PageRepository) List(ctx context.Context, opts page.ListOptions) ([]page.Page, int64, error) {
	var models []models.PageModel
	var total int64

	query := r.db.WithContext(ctx).Model(&models.PageModel{})

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	offset := (opts.Page - 1) * opts.PerPage
	if err := query.Offset(offset).Limit(opts.PerPage).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, 0, err
	}

	// Convert to domain types
	pages := make([]page.Page, len(models))
	for i, m := range models {
		pages[i] = *m.ToCore()
	}

	return pages, total, nil
}

// Save creates or updates a page
func (r *PageRepository) Save(ctx context.Context, p *page.Page) error {
	model := models.PageModelFromCore(p)

	// Check if page exists
	var existing models.PageModel
	err := r.db.WithContext(ctx).First(&existing, p.ID).Error
	if err == gorm.ErrRecordNotFound {
		// Create new page
		if p.ID == uuid.Nil {
			p.ID = uuid.New()
			model.ID = p.ID
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
	result := r.db.WithContext(ctx).Delete(&models.PageModel{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.ErrNotFound
	}
	return nil
}
