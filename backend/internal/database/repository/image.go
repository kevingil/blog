package repository

import (
	"context"

	"blog-agent-go/backend/internal/core"
	"blog-agent-go/backend/internal/core/image"
	"blog-agent-go/backend/internal/database/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ImageRepository implements image.ImageStore using GORM
type ImageRepository struct {
	db *gorm.DB
}

// NewImageRepository creates a new ImageRepository
func NewImageRepository(db *gorm.DB) *ImageRepository {
	return &ImageRepository{db: db}
}

// FindByID retrieves an image generation by its ID
func (r *ImageRepository) FindByID(ctx context.Context, id uuid.UUID) (*image.ImageGeneration, error) {
	var model models.ImageGenerationModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return model.ToCore(), nil
}

// FindByRequestID retrieves an image generation by its request ID
func (r *ImageRepository) FindByRequestID(ctx context.Context, requestID string) (*image.ImageGeneration, error) {
	var model models.ImageGenerationModel
	if err := r.db.WithContext(ctx).Where("request_id = ?", requestID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return model.ToCore(), nil
}

// Save creates a new image generation
func (r *ImageRepository) Save(ctx context.Context, i *image.ImageGeneration) error {
	model := models.ImageGenerationModelFromCore(i)

	if i.ID == uuid.Nil {
		i.ID = uuid.New()
		model.ID = i.ID
	}

	return r.db.WithContext(ctx).Create(model).Error
}

// Update updates an existing image generation
func (r *ImageRepository) Update(ctx context.Context, i *image.ImageGeneration) error {
	model := models.ImageGenerationModelFromCore(i)
	return r.db.WithContext(ctx).Save(model).Error
}
