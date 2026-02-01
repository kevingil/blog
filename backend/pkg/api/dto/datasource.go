package dto

import (
	"time"

	"github.com/google/uuid"
)

// DataSourceCreateRequest represents a request to create a data source
type DataSourceCreateRequest struct {
	Name           string  `json:"name" validate:"required,min=1,max=255"`
	URL            string  `json:"url" validate:"required,url"`
	FeedURL        *string `json:"feed_url" validate:"omitempty,url"`
	SourceType     string  `json:"source_type" validate:"omitempty,oneof=blog forum news rss newsletter"`
	CrawlFrequency string  `json:"crawl_frequency" validate:"omitempty,oneof=hourly daily weekly"`
	IsEnabled      *bool   `json:"is_enabled"`
}

// DataSourceUpdateRequest represents a request to update a data source
type DataSourceUpdateRequest struct {
	Name           *string `json:"name" validate:"omitempty,min=1,max=255"`
	URL            *string `json:"url" validate:"omitempty,url"`
	FeedURL        *string `json:"feed_url" validate:"omitempty,url"`
	SourceType     *string `json:"source_type" validate:"omitempty,oneof=blog forum news rss newsletter"`
	CrawlFrequency *string `json:"crawl_frequency" validate:"omitempty,oneof=hourly daily weekly"`
	IsEnabled      *bool   `json:"is_enabled"`
}

// DataSourceResponse is the response for a data source
type DataSourceResponse struct {
	ID               uuid.UUID              `json:"id"`
	OrganizationID   *uuid.UUID             `json:"organization_id,omitempty"`
	UserID           *uuid.UUID             `json:"user_id,omitempty"`
	Name             string                 `json:"name"`
	URL              string                 `json:"url"`
	FeedURL          *string                `json:"feed_url,omitempty"`
	SourceType       string                 `json:"source_type"`
	CrawlFrequency   string                 `json:"crawl_frequency"`
	IsEnabled        bool                   `json:"is_enabled"`
	IsDiscovered     bool                   `json:"is_discovered"`
	DiscoveredFromID *uuid.UUID             `json:"discovered_from_id,omitempty"`
	LastCrawledAt    *time.Time             `json:"last_crawled_at,omitempty"`
	NextCrawlAt      *time.Time             `json:"next_crawl_at,omitempty"`
	CrawlStatus      string                 `json:"crawl_status"`
	ErrorMessage     *string                `json:"error_message,omitempty"`
	ContentCount     int                    `json:"content_count"`
	SubscriberCount  int                    `json:"subscriber_count"`
	MetaData         map[string]interface{} `json:"meta_data,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// CrawledContentResponse is the response for crawled content
type CrawledContentResponse struct {
	ID           uuid.UUID              `json:"id"`
	DataSourceID uuid.UUID              `json:"data_source_id"`
	URL          string                 `json:"url"`
	Title        *string                `json:"title,omitempty"`
	Content      string                 `json:"content"`
	Summary      *string                `json:"summary,omitempty"`
	Author       *string                `json:"author,omitempty"`
	PublishedAt  *time.Time             `json:"published_at,omitempty"`
	MetaData     map[string]interface{} `json:"meta_data,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	// Joined fields
	DataSourceName *string `json:"data_source_name,omitempty"`
	DataSourceURL  *string `json:"data_source_url,omitempty"`
}
