package models

import (
	"encoding/json"
	"time"

	"backend/pkg/core/image"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// ImageGeneration is the GORM model for image generations
type ImageGeneration struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Prompt       string         `json:"prompt" gorm:"not null"`
	Provider     string         `json:"provider" gorm:"not null"`
	ModelName    string         `json:"model_name" gorm:"not null"`
	RequestID    string         `json:"request_id" gorm:"uniqueIndex"`
	Status       string         `json:"status" gorm:"default:'pending'"`
	OutputURL    string         `json:"output_url"`
	FileIndexID  *uuid.UUID     `json:"file_index_id" gorm:"type:uuid"`
	ErrorMessage string         `json:"error_message"`
	MetaData     datatypes.JSON `json:"meta_data" gorm:"type:jsonb;default:'{}'"`
	CreatedAt    time.Time      `json:"created_at" gorm:"autoCreateTime"`
	CompletedAt  *string        `json:"completed_at"`
}

func (ImageGeneration) TableName() string {
	return "imagen_request"
}

// ToCore converts the GORM model to the domain type
func (m *ImageGeneration) ToCore() *image.ImageGeneration {
	var metaData map[string]interface{}
	if m.MetaData != nil {
		_ = json.Unmarshal(m.MetaData, &metaData)
	}

	var completedAt *time.Time
	if m.CompletedAt != nil {
		if t, err := time.Parse(time.RFC3339, *m.CompletedAt); err == nil {
			completedAt = &t
		}
	}

	return &image.ImageGeneration{
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

// ImageGenerationFromCore creates a GORM model from the domain type
func ImageGenerationFromCore(i *image.ImageGeneration) *ImageGeneration {
	var metaData datatypes.JSON
	if i.MetaData != nil {
		metaData, _ = datatypes.NewJSONType(i.MetaData).MarshalJSON()
	}

	var completedAt *string
	if i.CompletedAt != nil {
		s := i.CompletedAt.Format(time.RFC3339)
		completedAt = &s
	}

	return &ImageGeneration{
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
