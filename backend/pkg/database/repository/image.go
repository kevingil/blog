package repository

import (
	"context"
	"encoding/json"
	"time"

	"backend/pkg/core"
	"backend/pkg/database/models"
	"backend/pkg/types"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ImageRepository defines the interface for image generation data access
type ImageRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*types.ImageGeneration, error)
	FindByRequestID(ctx context.Context, requestID string) (*types.ImageGeneration, error)
	Save(ctx context.Context, img *types.ImageGeneration) error
	Update(ctx context.Context, img *types.ImageGeneration) error
}

// imageRepository provides data access for image generations
type imageRepository struct {
	db *gorm.DB
}

// NewImageRepository creates a new ImageRepository
func NewImageRepository(db *gorm.DB) ImageRepository {
	return &imageRepository{db: db}
}

// imageModelToType converts a database model to types
func imageModelToType(m *models.ImageGeneration) *types.ImageGeneration {
	var metaData map[string]interface{}
	if m.MetaData != nil {
		_ = json.Unmarshal(m.MetaData, &metaData)
	}

	var completedAt *time.Time
	if m.CompletedAt != nil {
		t, err := time.Parse(time.RFC3339, *m.CompletedAt)
		if err == nil {
			completedAt = &t
		}
	}

	return &types.ImageGeneration{
		ID:           m.ID,
		Prompt:       m.Prompt,
		Provider:     m.Provider,
		ModelName:    m.ModelName,
		RequestID:    m.RequestID,
		Status:       m.Status,
		OutputURL:    m.OutputURL,
		FileIndexID:  m.FileIndexID,
		ErrorMessage: m.ErrorMessage,
		MetaData:     metaData,
		CreatedAt:    m.CreatedAt,
		CompletedAt:  completedAt,
	}
}

// imageTypeToModel converts a types type to database model
func imageTypeToModel(i *types.ImageGeneration) *models.ImageGeneration {
	var metaData datatypes.JSON
	if i.MetaData != nil {
		data, _ := json.Marshal(i.MetaData)
		metaData = datatypes.JSON(data)
	}

	var completedAt *string
	if i.CompletedAt != nil {
		s := i.CompletedAt.Format(time.RFC3339)
		completedAt = &s
	}

	return &models.ImageGeneration{
		ID:           i.ID,
		Prompt:       i.Prompt,
		Provider:     i.Provider,
		ModelName:    i.ModelName,
		RequestID:    i.RequestID,
		Status:       i.Status,
		OutputURL:    i.OutputURL,
		FileIndexID:  i.FileIndexID,
		ErrorMessage: i.ErrorMessage,
		MetaData:     metaData,
		CreatedAt:    i.CreatedAt,
		CompletedAt:  completedAt,
	}
}

// FindByID retrieves an image generation by its ID
func (r *imageRepository) FindByID(ctx context.Context, id uuid.UUID) (*types.ImageGeneration, error) {
	var model models.ImageGeneration
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return imageModelToType(&model), nil
}

// FindByRequestID retrieves an image generation by its request ID
func (r *imageRepository) FindByRequestID(ctx context.Context, requestID string) (*types.ImageGeneration, error) {
	var model models.ImageGeneration
	if err := r.db.WithContext(ctx).Where("request_id = ?", requestID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return imageModelToType(&model), nil
}

// Save creates a new image generation
func (r *imageRepository) Save(ctx context.Context, img *types.ImageGeneration) error {
	model := imageTypeToModel(img)
	if model.ID == uuid.Nil {
		model.ID = uuid.New()
		img.ID = model.ID
	}
	return r.db.WithContext(ctx).Create(model).Error
}

// Update updates an existing image generation
func (r *imageRepository) Update(ctx context.Context, img *types.ImageGeneration) error {
	model := imageTypeToModel(img)
	return r.db.WithContext(ctx).Save(model).Error
}
