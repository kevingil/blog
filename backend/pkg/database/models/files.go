package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type FileIndex struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	S3Key         string         `json:"s3_key" gorm:"not null;uniqueIndex"`
	Filename      string         `json:"filename" gorm:"not null"`
	DirectoryPath string         `json:"directory_path"`
	FileType      string         `json:"file_type"`
	FileSize      int64          `json:"file_size"`
	ContentType   string         `json:"content_type"`
	MetaData      datatypes.JSON `json:"meta_data" gorm:"type:jsonb;default:'{}'"`
	CreatedAt     time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
}

func (FileIndex) TableName() string {
	return "file_index"
}
