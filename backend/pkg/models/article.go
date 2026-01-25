package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
	"gorm.io/datatypes"
)

// Article represents a blog article with cached draft and published content
type Article struct {
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

// IsPublished returns true if the article has been published
func (a *Article) IsPublished() bool {
	return a.PublishedAt != nil
}

// GetTitle returns the appropriate title based on context
// For public viewing, returns published title if available
// For editing, returns draft title
func (a *Article) GetTitle(forEditing bool) string {
	if forEditing || a.PublishedTitle == nil {
		return a.DraftTitle
	}
	return *a.PublishedTitle
}

// GetContent returns the appropriate content based on context
func (a *Article) GetContent(forEditing bool) string {
	if forEditing || a.PublishedContent == nil {
		return a.DraftContent
	}
	return *a.PublishedContent
}

// GetImageURL returns the appropriate image URL based on context
func (a *Article) GetImageURL(forEditing bool) string {
	if forEditing || a.PublishedImageURL == nil {
		return a.DraftImageURL
	}
	return *a.PublishedImageURL
}

// ArticleVersion represents a historical snapshot of an article
type ArticleVersion struct {
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

func (ArticleVersion) TableName() string {
	return "article_version"
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
	ID         uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	ArticleID  uuid.UUID       `json:"article_id" gorm:"type:uuid;not null;index"`
	Title      string          `json:"title"`
	Content    string          `json:"content" gorm:"type:text;not null"`
	URL        string          `json:"url"`
	SourceType string          `json:"source_type" gorm:"default:web"`
	Embedding  pgvector.Vector `json:"embedding" gorm:"type:vector(1536)"`
	MetaData   datatypes.JSON  `json:"meta_data" gorm:"type:jsonb;default:'{}'"`
	CreatedAt  time.Time       `json:"created_at" gorm:"autoCreateTime"`
}

func (ArticleSource) TableName() string {
	return "article_source"
}
