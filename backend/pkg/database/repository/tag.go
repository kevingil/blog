package repository

import (
	"context"
	"strings"

	"backend/pkg/core"
	"backend/pkg/core/tag"
	"backend/pkg/database/models"

	"gorm.io/gorm"
)

// TagRepository implements tag.TagStore using GORM
type TagRepository struct {
	db *gorm.DB
}

// NewTagRepository creates a new TagRepository
func NewTagRepository(db *gorm.DB) *TagRepository {
	return &TagRepository{db: db}
}

// FindByID retrieves a tag by its ID
func (r *TagRepository) FindByID(ctx context.Context, id int) (*tag.Tag, error) {
	var model models.Tag
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return model.ToCore(), nil
}

// FindByName retrieves a tag by its name (case-insensitive)
func (r *TagRepository) FindByName(ctx context.Context, name string) (*tag.Tag, error) {
	var model models.Tag
	if err := r.db.WithContext(ctx).Where("LOWER(name) = LOWER(?)", name).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return model.ToCore(), nil
}

// FindByIDs retrieves tags by their IDs
func (r *TagRepository) FindByIDs(ctx context.Context, ids []int64) ([]tag.Tag, error) {
	var tagModels []models.Tag
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&tagModels).Error; err != nil {
		return nil, err
	}

	tags := make([]tag.Tag, len(tagModels))
	for i, m := range tagModels {
		tags[i] = *m.ToCore()
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
func (r *TagRepository) List(ctx context.Context) ([]tag.Tag, error) {
	var tagModels []models.Tag
	if err := r.db.WithContext(ctx).Order("name ASC").Find(&tagModels).Error; err != nil {
		return nil, err
	}

	tags := make([]tag.Tag, len(tagModels))
	for i, m := range tagModels {
		tags[i] = *m.ToCore()
	}
	return tags, nil
}

// Save creates or updates a tag
func (r *TagRepository) Save(ctx context.Context, t *tag.Tag) error {
	model := models.TagFromCore(t)

	if t.ID == 0 {
		// Create new tag
		if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
			return err
		}
		t.ID = model.ID
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
