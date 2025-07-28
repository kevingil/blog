package models

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

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
	CreatedAt    string         `json:"created_at" gorm:"autoCreateTime"`
	CompletedAt  *string        `json:"completed_at"`
}

func (ImageGeneration) TableName() string {
	return "imagen_request"
}

type FileIndex struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	S3Key         string         `json:"s3_key" gorm:"not null;uniqueIndex"`
	Filename      string         `json:"filename" gorm:"not null"`
	DirectoryPath string         `json:"directory_path"`
	FileType      string         `json:"file_type"`
	FileSize      int64          `json:"file_size"`
	ContentType   string         `json:"content_type"`
	MetaData      datatypes.JSON `json:"meta_data" gorm:"type:jsonb;default:'{}'"`
	CreatedAt     string         `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     string         `json:"updated_at" gorm:"autoUpdateTime"`
}

func (FileIndex) TableName() string {
	return "file_index"
}
