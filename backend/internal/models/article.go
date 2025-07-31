package models

import (
	"encoding/json"
	"time"

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
	CreatedAt       time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	PublishedAt     *time.Time     `json:"published_at,omitempty"`
	ImagenRequestID *uuid.UUID     `json:"imagen_request_id" gorm:"type:uuid"`
	Embedding       []float32      `json:"embedding" gorm:"type:vector(1536)"`
	SessionMemory   datatypes.JSON `json:"session_memory" gorm:"type:jsonb;default:'{}'"`
}

func (Article) TableName() string {
	return "article"
}

func (a Article) MarshalJSON() ([]byte, error) {
	type Alias Article
	aux := struct {
		PublishedAt *string `json:"published_at,omitempty"`
		Alias
	}{
		Alias: (Alias)(a),
	}
	if a.PublishedAt != nil {
		year := a.PublishedAt.Year()
		if year >= 0 && year <= 9999 {
			s := a.PublishedAt.UTC().Format(time.RFC3339)
			aux.PublishedAt = &s
		} else {
			aux.PublishedAt = nil
		}
	}
	return json.Marshal(aux)
}

type Tag struct {
	ID        int       `json:"id" gorm:"primaryKey;autoIncrement"`
	Name      string    `json:"name" gorm:"uniqueIndex;not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (Tag) TableName() string {
	return "tag"
}

// ArticleSource represents a source/citation for an article
// Matches the article_source table

type ArticleSource struct {
	ID         uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	ArticleID  uuid.UUID      `json:"article_id" gorm:"type:uuid;not null;index"`
	Title      string         `json:"title"`
	Content    string         `json:"content" gorm:"type:text;not null"`
	URL        string         `json:"url"`
	SourceType string         `json:"source_type" gorm:"default:web"`
	Embedding  []float32      `json:"embedding" gorm:"type:vector(1536)"`
	MetaData   datatypes.JSON `json:"meta_data" gorm:"type:jsonb;default:'{}'"`
	CreatedAt  time.Time      `json:"created_at" gorm:"autoCreateTime"`
}

func (ArticleSource) TableName() string {
	return "article_source"
}
