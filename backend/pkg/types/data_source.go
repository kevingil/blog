package types

import (
	"time"

	"github.com/google/uuid"
)

// DataSource represents a user's preferred website to crawl
type DataSource struct {
	ID               uuid.UUID
	OrganizationID   *uuid.UUID
	UserID           *uuid.UUID
	Name             string
	URL              string
	FeedURL          *string
	SourceType       string
	CrawlFrequency   string
	IsEnabled        bool
	IsDiscovered     bool
	DiscoveredFromID *uuid.UUID
	LastCrawledAt    *time.Time
	NextCrawlAt      *time.Time
	CrawlStatus      string
	ErrorMessage     *string
	ContentCount     int
	SubscriberCount  int
	MetaData         map[string]interface{}
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// CrawledContent represents content fetched from a data source
type CrawledContent struct {
	ID           uuid.UUID
	DataSourceID uuid.UUID
	URL          string
	Title        *string
	Content      string
	Summary      *string
	Author       *string
	PublishedAt  *time.Time
	Embedding    []float32
	MetaData     map[string]interface{}
	CreatedAt    time.Time
}
