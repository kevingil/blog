package models

import (
	"encoding/json"
	"time"

	"backend/pkg/core/article"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
	"gorm.io/datatypes"
)

// ArticleModel is the GORM model for articles
type ArticleModel struct {
	ID              uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Slug            string          `json:"slug" gorm:"uniqueIndex;not null"`
	Title           string          `json:"title" gorm:"not null"`
	Content         string          `json:"content" gorm:"type:text"`
	ImageURL        string          `json:"image_url"`
	AuthorID        uuid.UUID       `json:"author_id" gorm:"type:uuid;not null"`
	TagIDs          pq.Int64Array   `json:"tag_ids" gorm:"type:integer[]"`
	IsDraft         bool            `json:"is_draft" gorm:"default:true"`
	CreatedAt       time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
	PublishedAt     *time.Time      `json:"published_at,omitempty"`
	ImagenRequestID *uuid.UUID      `json:"imagen_request_id" gorm:"type:uuid"`
	Embedding       pgvector.Vector `json:"embedding" gorm:"type:vector(1536)"`
	SessionMemory   datatypes.JSON  `json:"session_memory" gorm:"type:jsonb;default:'{}'"`
}

func (ArticleModel) TableName() string {
	return "article"
}

func (a ArticleModel) MarshalJSON() ([]byte, error) {
	type Alias ArticleModel
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

// ToCore converts the GORM model to the domain type
func (m *ArticleModel) ToCore() *article.Article {
	var sessionMemory map[string]interface{}
	if m.SessionMemory != nil {
		_ = json.Unmarshal(m.SessionMemory, &sessionMemory)
	}

	var embedding []float32
	if m.Embedding.Slice() != nil {
		embedding = m.Embedding.Slice()
	}

	return &article.Article{
		ID:              m.ID,
		Slug:            m.Slug,
		Title:           m.Title,
		Content:         m.Content,
		ImageURL:        m.ImageURL,
		AuthorID:        m.AuthorID,
		TagIDs:          m.TagIDs,
		IsDraft:         m.IsDraft,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
		PublishedAt:     m.PublishedAt,
		ImagenRequestID: m.ImagenRequestID,
		Embedding:       embedding,
		SessionMemory:   sessionMemory,
	}
}

// ArticleModelFromCore creates a GORM model from the domain type
func ArticleModelFromCore(a *article.Article) *ArticleModel {
	var sessionMemory datatypes.JSON
	if a.SessionMemory != nil {
		sessionMemory, _ = datatypes.NewJSONType(a.SessionMemory).MarshalJSON()
	}

	var embedding pgvector.Vector
	if len(a.Embedding) > 0 {
		embedding = pgvector.NewVector(a.Embedding)
	}

	return &ArticleModel{
		ID:              a.ID,
		Slug:            a.Slug,
		Title:           a.Title,
		Content:         a.Content,
		ImageURL:        a.ImageURL,
		AuthorID:        a.AuthorID,
		TagIDs:          a.TagIDs,
		IsDraft:         a.IsDraft,
		CreatedAt:       a.CreatedAt,
		UpdatedAt:       a.UpdatedAt,
		PublishedAt:     a.PublishedAt,
		ImagenRequestID: a.ImagenRequestID,
		Embedding:       embedding,
		SessionMemory:   sessionMemory,
	}
}
