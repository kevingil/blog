package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"gorm.io/datatypes"
)

// Source is the GORM model for article sources
type Source struct {
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

func (Source) TableName() string {
	return "article_source"
}
