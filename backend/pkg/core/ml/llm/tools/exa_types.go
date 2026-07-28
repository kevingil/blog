package tools

import (
	"time"

	"backend/pkg/database/models"

	"github.com/google/uuid"
)

// WebContentSource represents a web-based content source with full text
// This type is specifically designed for search results that include complete webpage content
type WebContentSource struct {
	ID            uuid.UUID `json:"id"`
	ArticleID     uuid.UUID `json:"article_id"`
	Title         string    `json:"title"`
	URL           string    `json:"url"`
	FullText      string    `json:"full_text"`      // Complete webpage text content
	Summary       string    `json:"summary"`        // Optional summary
	Author        string    `json:"author"`         // Optional author information
	PublishedDate string    `json:"published_date"` // Optional publication date
	Highlights    []string  `json:"highlights"`     // Key highlights from the content
	SourceType    string    `json:"source_type"`    // Type of source (e.g., "web_search", "manual")
	SearchQuery   string    `json:"search_query"`   // The original search query that found this source
	Metadata      Metadata  `json:"metadata"`       // Additional metadata
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Metadata contains additional information about the web content source
type Metadata struct {
	SearchEngine   string                 `json:"search_engine"`   // e.g., "exa", "google"
	SearchID       string                 `json:"search_id"`       // Original search result ID
	Score          float64                `json:"score"`           // Relevance score
	Favicon        string                 `json:"favicon"`         // Website favicon URL
	Image          string                 `json:"image"`           // Associated image URL
	WordCount      int                    `json:"word_count"`      // Approximate word count
	Language       string                 `json:"language"`        // Detected language
	ContentQuality string                 `json:"content_quality"` // Quality assessment
	Extras         map[string]interface{} `json:"extras"`          // Additional provider-specific data
}

// ToSource converts a WebContentSource to a Source for database storage
func (wcs *WebContentSource) ToSource() *models.Source {
	// Combine full text and summary for content
	content := wcs.FullText
	if wcs.Summary != "" && wcs.Summary != wcs.FullText {
		content = wcs.Summary + "\n\n" + wcs.FullText
	}

	// Note: Metadata will be handled by the service layer when creating the Source

	return &models.Source{
		ID:         wcs.ID,
		ArticleID:  wcs.ArticleID,
		Title:      wcs.Title,
		Content:    content,
		URL:        wcs.URL,
		SourceType: wcs.SourceType,
		// Note: Embedding will be generated separately when saving to database
		// MetaData: datatypes.JSON(metadataJSON), // Will be set by the service
		CreatedAt: wcs.CreatedAt,
	}
}

// NewWebContentSourceFromSearchResult creates a WebContentSource from search result data
func NewWebContentSourceFromSearchResult(articleID uuid.UUID, searchQuery string, result SearchResultData) *WebContentSource {
	return &WebContentSource{
		ID:            uuid.New(),
		ArticleID:     articleID,
		Title:         result.Title,
		URL:           result.URL,
		FullText:      result.Text,
		Summary:       result.Summary,
		Author:        result.Author,
		PublishedDate: result.PublishedDate,
		Highlights:    result.Highlights,
		SourceType:    "web_search",
		SearchQuery:   searchQuery,
		Metadata: Metadata{
			SearchEngine:   "exa",
			SearchID:       result.ID,
			Score:          result.Score,
			Favicon:        result.Favicon,
			Image:          result.Image,
			WordCount:      len(result.Text) / 5, // Rough word count estimate
			Language:       "en",                 // Default to English, could be detected
			ContentQuality: "high",               // Default quality assessment
			Extras:         result.Extras,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// SearchResultData represents the structure of search result data
// This is used to create WebContentSource instances from various search providers
type SearchResultData struct {
	ID            string                 `json:"id"`
	Title         string                 `json:"title"`
	URL           string                 `json:"url"`
	Text          string                 `json:"text"`
	Summary       string                 `json:"summary"`
	Author        string                 `json:"author"`
	PublishedDate string                 `json:"published_date"`
	Highlights    []string               `json:"highlights"`
	Score         float64                `json:"score"`
	Favicon       string                 `json:"favicon"`
	Image         string                 `json:"image"`
	Extras        map[string]interface{} `json:"extras"`
}
