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

// SourceListOptions represents options for listing sources
type SourceListOptions struct {
	Page    int
	PerPage int
}

// SourceWithArticle includes article metadata with the source
type SourceWithArticle struct {
	Source       Source
	ArticleTitle string
	ArticleSlug  string
}
