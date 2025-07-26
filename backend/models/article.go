package models

import (
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/datatypes"
)

type Article struct {
	ID              uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Slug            string         `json:"slug" gorm:"uniqueIndex;not null"`
	Title           string         `json:"title" gorm:"not null"`
	Content         string         `json:"content" gorm:"type:text"`
	ImageURL        string         `json:"image_url"`
	AuthorID        uuid.UUID      `json:"author_id" gorm:"type:uuid;not null"`
	TagIDs          pq.Int64Array  `json:"tag_ids" gorm:"type:integer[]"`
	IsDraft         bool           `json:"is_draft" gorm:"default:true"`
	CreatedAt       string         `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       string         `json:"updated_at" gorm:"autoUpdateTime"`
	PublishedAt     *string        `json:"published_at,omitempty"`
	ImagenRequestID *uuid.UUID     `json:"imagen_request_id" gorm:"type:uuid"`
	Embedding       []float32      `json:"embedding" gorm:"type:vector(1536)"`
	SessionMemory   datatypes.JSON `json:"session_memory" gorm:"type:jsonb;default:'{}'"`
}

func (Article) TableName() string {
	return "article"
}

type Tag struct {
	ID        int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Name      string `json:"name" gorm:"uniqueIndex;not null"`
	CreatedAt string `json:"created_at" gorm:"autoCreateTime"`
}

func (Tag) TableName() string {
	return "tag"
}
