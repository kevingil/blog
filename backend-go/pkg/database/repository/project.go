package repository

import (
	"context"

	"backend/pkg/core"
	"backend/pkg/database/models"
	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// ProjectRepository defines the interface for project data access
type ProjectRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*types.Project, error)
	List(ctx context.Context, opts types.ProjectListOptions) ([]types.Project, int64, error)
	Save(ctx context.Context, project *types.Project) error
	Update(ctx context.Context, project *types.Project) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// projectRepository implements data access for projects using GORM
type projectRepository struct {
	db *gorm.DB
}

// NewProjectRepository creates a new ProjectRepository
func NewProjectRepository(db *gorm.DB) ProjectRepository {
	return &projectRepository{db: db}
}

// projectModelToType converts a GORM model to the types
func projectModelToType(m *models.Project) *types.Project {
	return &types.Project{
		ID:          m.ID,
		Title:       m.Title,
		Description: m.Description,
		Content:     m.Content,
		TagIDs:      m.TagIDs,
		ImageURL:    m.ImageURL,
		URL:         m.URL,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// projectTypeToModel creates a GORM model from the types
func projectTypeToModel(p *types.Project) *models.Project {
	return &models.Project{
		ID:          p.ID,
		Title:       p.Title,
		Description: p.Description,
		Content:     p.Content,
		TagIDs:      pq.Int64Array(p.TagIDs),
		ImageURL:    p.ImageURL,
		URL:         p.URL,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// FindByID retrieves a project by its ID
func (r *projectRepository) FindByID(ctx context.Context, id uuid.UUID) (*types.Project, error) {
	var model models.Project
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return projectModelToType(&model), nil
}

// List retrieves projects with pagination
func (r *projectRepository) List(ctx context.Context, opts types.ProjectListOptions) ([]types.Project, int64, error) {
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

	// Convert to types
	projects := make([]types.Project, len(projectModels))
	for i, m := range projectModels {
		projects[i] = *projectModelToType(&m)
	}

	return projects, total, nil
}

// Save creates a new project
func (r *projectRepository) Save(ctx context.Context, p *types.Project) error {
	model := projectTypeToModel(p)

	if p.ID == uuid.Nil {
		p.ID = uuid.New()
		model.ID = p.ID
	}

	return r.db.WithContext(ctx).Create(model).Error
}

// Update updates an existing project
func (r *projectRepository) Update(ctx context.Context, p *types.Project) error {
	model := projectTypeToModel(p)
	return r.db.WithContext(ctx).Save(model).Error
}

// Delete removes a project by its ID
func (r *projectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Project{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.ErrNotFound
	}
	return nil
}
