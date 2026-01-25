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

// ArticleModel is the GORM model for articles with cached draft/published content
type ArticleModel struct {
	ID       uuid.UUID     `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Slug     string        `json:"slug" gorm:"uniqueIndex;not null"`
	AuthorID uuid.UUID     `json:"author_id" gorm:"type:uuid;not null"`
	TagIDs   pq.Int64Array `json:"tag_ids" gorm:"type:integer[]"`

	// Cached draft content
	DraftTitle     string          `json:"draft_title"`
	DraftContent   string          `json:"draft_content" gorm:"type:text"`
	DraftImageURL  string          `json:"draft_image_url"`
	DraftEmbedding pgvector.Vector `json:"draft_embedding" gorm:"type:vector(1536)"`

	// Cached published content
	PublishedTitle     *string         `json:"published_title"`
	PublishedContent   *string         `json:"published_content" gorm:"type:text"`
	PublishedImageURL  *string         `json:"published_image_url"`
	PublishedEmbedding pgvector.Vector `json:"published_embedding" gorm:"type:vector(1536)"`
	PublishedAt        *time.Time      `json:"published_at,omitempty"`

	// Version pointers
	CurrentDraftVersionID     *uuid.UUID `json:"current_draft_version_id" gorm:"type:uuid"`
	CurrentPublishedVersionID *uuid.UUID `json:"current_published_version_id" gorm:"type:uuid"`

	// Metadata
	ImagenRequestID *uuid.UUID     `json:"imagen_request_id" gorm:"type:uuid"`
	SessionMemory   datatypes.JSON `json:"session_memory" gorm:"type:jsonb;default:'{}'"`
	CreatedAt       time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
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

	var draftEmbedding []float32
	if m.DraftEmbedding.Slice() != nil {
		draftEmbedding = m.DraftEmbedding.Slice()
	}

	var publishedEmbedding []float32
	if m.PublishedEmbedding.Slice() != nil {
		publishedEmbedding = m.PublishedEmbedding.Slice()
	}

	return &article.Article{
		ID:                        m.ID,
		Slug:                      m.Slug,
		AuthorID:                  m.AuthorID,
		TagIDs:                    m.TagIDs,
		DraftTitle:                m.DraftTitle,
		DraftContent:              m.DraftContent,
		DraftImageURL:             m.DraftImageURL,
		DraftEmbedding:            draftEmbedding,
		PublishedTitle:            m.PublishedTitle,
		PublishedContent:          m.PublishedContent,
		PublishedImageURL:         m.PublishedImageURL,
		PublishedEmbedding:        publishedEmbedding,
		PublishedAt:               m.PublishedAt,
		CurrentDraftVersionID:     m.CurrentDraftVersionID,
		CurrentPublishedVersionID: m.CurrentPublishedVersionID,
		ImagenRequestID:           m.ImagenRequestID,
		SessionMemory:             sessionMemory,
		CreatedAt:                 m.CreatedAt,
		UpdatedAt:                 m.UpdatedAt,
	}
}

// ArticleModelFromCore creates a GORM model from the domain type
func ArticleModelFromCore(a *article.Article) *ArticleModel {
	var sessionMemory datatypes.JSON
	if a.SessionMemory != nil {
		sessionMemory, _ = datatypes.NewJSONType(a.SessionMemory).MarshalJSON()
	}

	var draftEmbedding pgvector.Vector
	if len(a.DraftEmbedding) > 0 {
		draftEmbedding = pgvector.NewVector(a.DraftEmbedding)
	}

	var publishedEmbedding pgvector.Vector
	if len(a.PublishedEmbedding) > 0 {
		publishedEmbedding = pgvector.NewVector(a.PublishedEmbedding)
	}

	return &ArticleModel{
		ID:                        a.ID,
		Slug:                      a.Slug,
		AuthorID:                  a.AuthorID,
		TagIDs:                    a.TagIDs,
		DraftTitle:                a.DraftTitle,
		DraftContent:              a.DraftContent,
		DraftImageURL:             a.DraftImageURL,
		DraftEmbedding:            draftEmbedding,
		PublishedTitle:            a.PublishedTitle,
		PublishedContent:          a.PublishedContent,
		PublishedImageURL:         a.PublishedImageURL,
		PublishedEmbedding:        publishedEmbedding,
		PublishedAt:               a.PublishedAt,
		CurrentDraftVersionID:     a.CurrentDraftVersionID,
		CurrentPublishedVersionID: a.CurrentPublishedVersionID,
		ImagenRequestID:           a.ImagenRequestID,
		SessionMemory:             sessionMemory,
		CreatedAt:                 a.CreatedAt,
		UpdatedAt:                 a.UpdatedAt,
	}
}

// ArticleVersionModel is the GORM model for article version history
type ArticleVersionModel struct {
	ID            uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	ArticleID     uuid.UUID       `json:"article_id" gorm:"type:uuid;not null;index"`
	VersionNumber int             `json:"version_number" gorm:"not null"`
	Status        string          `json:"status" gorm:"type:varchar(20);not null"`
	Title         string          `json:"title" gorm:"type:varchar(500);not null"`
	Content       string          `json:"content" gorm:"type:text"`
	ImageURL      string          `json:"image_url"`
	Embedding     pgvector.Vector `json:"embedding" gorm:"type:vector(1536)"`
	EditedBy      *uuid.UUID      `json:"edited_by" gorm:"type:uuid"`
	CreatedAt     time.Time       `json:"created_at" gorm:"autoCreateTime"`
}

func (ArticleVersionModel) TableName() string {
	return "article_version"
}

// ToCore converts the GORM model to the domain type
func (m *ArticleVersionModel) ToCore() *article.ArticleVersion {
	var embedding []float32
	if m.Embedding.Slice() != nil {
		embedding = m.Embedding.Slice()
	}

	return &article.ArticleVersion{
		ID:            m.ID,
		ArticleID:     m.ArticleID,
		VersionNumber: m.VersionNumber,
		Status:        m.Status,
		Title:         m.Title,
		Content:       m.Content,
		ImageURL:      m.ImageURL,
		Embedding:     embedding,
		EditedBy:      m.EditedBy,
		CreatedAt:     m.CreatedAt,
	}
}

// ArticleVersionModelFromCore creates a GORM model from the domain type
func ArticleVersionModelFromCore(v *article.ArticleVersion) *ArticleVersionModel {
	var embedding pgvector.Vector
	if len(v.Embedding) > 0 {
		embedding = pgvector.NewVector(v.Embedding)
	}

	return &ArticleVersionModel{
		ID:            v.ID,
		ArticleID:     v.ArticleID,
		VersionNumber: v.VersionNumber,
		Status:        v.Status,
		Title:         v.Title,
		Content:       v.Content,
		ImageURL:      v.ImageURL,
		Embedding:     embedding,
		EditedBy:      v.EditedBy,
		CreatedAt:     v.CreatedAt,
	}
}
