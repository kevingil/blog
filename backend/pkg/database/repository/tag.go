package repository

import (
	"context"
	"strings"

	"backend/pkg/core"
	"backend/pkg/database/models"
	"backend/pkg/types"

	"gorm.io/gorm"
)

// TagRepository provides data access for tags
type TagRepository struct {
	db *gorm.DB
}

// NewTagRepository creates a new TagRepository
func NewTagRepository(db *gorm.DB) *TagRepository {
	return &TagRepository{db: db}
}

// tagModelToType converts a database model to types
func tagModelToType(m *models.Tag) *types.Tag {
	return &types.Tag{
		ID:        m.ID,
		Name:      m.Name,
		CreatedAt: m.CreatedAt,
	}
}

// tagTypeToModel converts a types type to database model
func tagTypeToModel(t *types.Tag) *models.Tag {
	return &models.Tag{
		ID:        t.ID,
		Name:      t.Name,
		CreatedAt: t.CreatedAt,
	}
}

// FindByID retrieves a tag by its ID
func (r *TagRepository) FindByID(ctx context.Context, id int) (*types.Tag, error) {
	var model models.Tag
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return tagModelToType(&model), nil
}

// FindByName retrieves a tag by its name (case-insensitive)
func (r *TagRepository) FindByName(ctx context.Context, name string) (*types.Tag, error) {
	var model models.Tag
	if err := r.db.WithContext(ctx).Where("LOWER(name) = LOWER(?)", name).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return tagModelToType(&model), nil
}

// FindByIDs retrieves tags by their IDs
func (r *TagRepository) FindByIDs(ctx context.Context, ids []int64) ([]types.Tag, error) {
	var tagModels []models.Tag
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&tagModels).Error; err != nil {
		return nil, err
	}

	tags := make([]types.Tag, len(tagModels))
	for i, m := range tagModels {
		tags[i] = *tagModelToType(&m)
	}
	return tags, nil
}

// EnsureExists creates tags if they don't exist and returns their IDs
func (r *TagRepository) EnsureExists(ctx context.Context, names []string) ([]int64, error) {
	tagIDs := make([]int64, 0, len(names))

	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		var existingTag models.Tag
		err := r.db.WithContext(ctx).Where("LOWER(name) = LOWER(?)", name).First(&existingTag).Error
		if err == gorm.ErrRecordNotFound {
			// Create new tag
			newTag := &models.Tag{Name: name}
			if err := r.db.WithContext(ctx).Create(newTag).Error; err != nil {
				return nil, err
			}
			tagIDs = append(tagIDs, int64(newTag.ID))
		} else if err != nil {
			return nil, err
		} else {
			tagIDs = append(tagIDs, int64(existingTag.ID))
		}
	}

	return tagIDs, nil
}

// List retrieves all tags
func (r *TagRepository) List(ctx context.Context) ([]types.Tag, error) {
	var tagModels []models.Tag
	if err := r.db.WithContext(ctx).Order("name ASC").Find(&tagModels).Error; err != nil {
		return nil, err
	}

	tags := make([]types.Tag, len(tagModels))
	for i, m := range tagModels {
		tags[i] = *tagModelToType(&m)
	}
	return tags, nil
}

// Save creates or updates a tag
func (r *TagRepository) Save(ctx context.Context, tag *types.Tag) error {
	model := tagTypeToModel(tag)
	if model.ID == 0 {
		// Create new tag
		if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
			return err
		}
		tag.ID = model.ID
		return nil
	}

	// Update existing tag
	return r.db.WithContext(ctx).Save(model).Error
}

// Delete removes a tag by its ID
func (r *TagRepository) Delete(ctx context.Context, id int) error {
	result := r.db.WithContext(ctx).Delete(&models.Tag{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.ErrNotFound
	}
	return nil
}
