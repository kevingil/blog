package types

import (
	"time"

	"github.com/google/uuid"
)

// Source represents a source/citation for an article
type Source struct {
	ID         uuid.UUID
	ArticleID  uuid.UUID
	Title      string
	Content    string
	URL        string
	SourceType string
	Embedding  []float32
	MetaData   map[string]interface{}
	CreatedAt  time.Time
}
