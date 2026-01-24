package models

import (
	"time"

	"blog-agent-go/backend/internal/core/source"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"gorm.io/datatypes"
)

// SourceModel is the GORM model for article sources
type SourceModel struct {
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

func (SourceModel) TableName() string {
	return "article_source"
}

// ToCore converts the GORM model to the domain type
func (m *SourceModel) ToCore() *source.Source {
	var metaData map[string]interface{}
	if m.MetaData != nil {
		_ = m.MetaData.Unmarshal(&metaData)
	}

	var embedding []float32
	if m.Embedding.Slice() != nil {
		embedding = m.Embedding.Slice()
	}

	return &source.Source{
		ID:         m.ID,
		ArticleID:  m.ArticleID,
		Title:      m.Title,
		Content:    m.Content,
		URL:        m.URL,
		SourceType: m.SourceType,
		Embedding:  embedding,
		MetaData:   metaData,
		CreatedAt:  m.CreatedAt,
	}
}

// SourceModelFromCore creates a GORM model from the domain type
func SourceModelFromCore(s *source.Source) *SourceModel {
	var metaData datatypes.JSON
	if s.MetaData != nil {
		metaData, _ = datatypes.NewJSONType(s.MetaData).MarshalJSON()
	}

	var embedding pgvector.Vector
	if len(s.Embedding) > 0 {
		embedding = pgvector.NewVector(s.Embedding)
	}

	return &SourceModel{
		ID:         s.ID,
		ArticleID:  s.ArticleID,
		Title:      s.Title,
		Content:    s.Content,
		URL:        s.URL,
		SourceType: s.SourceType,
		Embedding:  embedding,
		MetaData:   metaData,
		CreatedAt:  s.CreatedAt,
	}
}
