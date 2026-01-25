package repository

import (
	"context"

	"backend/pkg/core"
	"backend/pkg/core/project"
	"backend/pkg/database/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProjectRepository implements project.ProjectStore using GORM
type ProjectRepository struct {
	db *gorm.DB
}

// NewProjectRepository creates a new ProjectRepository
func NewProjectRepository(db *gorm.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// FindByID retrieves a project by its ID
func (r *ProjectRepository) FindByID(ctx context.Context, id uuid.UUID) (*project.Project, error) {
	var model models.Project
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return model.ToCore(), nil
}

// List retrieves projects with pagination
func (r *ProjectRepository) List(ctx context.Context, opts project.ListOptions) ([]project.Project, int64, error) {
	var projectModels []models.Project
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Project{})

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	offset := (opts.Page - 1) * opts.PerPage
	if err := query.Offset(offset).Limit(opts.PerPage).Order("created_at DESC").Find(&projectModels).Error; err != nil {
		return nil, 0, err
	}

	// Convert to domain types
	projects := make([]project.Project, len(projectModels))
	for i, m := range projectModels {
		projects[i] = *m.ToCore()
	}

	return projects, total, nil
}

// Save creates a new project
func (r *ProjectRepository) Save(ctx context.Context, p *project.Project) error {
	model := models.ProjectFromCore(p)

	if p.ID == uuid.Nil {
		p.ID = uuid.New()
		model.ID = p.ID
	}

	return r.db.WithContext(ctx).Create(model).Error
}

// Update updates an existing project
func (r *ProjectRepository) Update(ctx context.Context, p *project.Project) error {
	model := models.ProjectFromCore(p)
	return r.db.WithContext(ctx).Save(model).Error
}

// Delete removes a project by its ID
func (r *ProjectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Project{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.ErrNotFound
	}
	return nil
}
