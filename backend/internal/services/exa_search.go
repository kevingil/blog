package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// ExaSearchService handles searching the web using Exa API
type ExaSearchService struct {
	apiKey string
	client *http.Client
}

// NewExaSearchService creates a new Exa search service
func NewExaSearchService() *ExaSearchService {
	apiKey := os.Getenv("EXA_API_KEY")
	if apiKey == "" {
		// This is not fatal - the tool will handle missing API key gracefully
		return &ExaSearchService{
			client: &http.Client{Timeout: 30 * time.Second},
		}
	}

	return &ExaSearchService{
		apiKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// ExaSearchRequest represents the request payload for Exa search
type ExaSearchRequest struct {
	Query              string   `json:"query"`
	Type               string   `json:"type,omitempty"`               // "neural", "keyword", or "auto" (default)
	UseAutoprompt      bool     `json:"useAutoprompt,omitempty"`      // Whether to use Exa's autoprompt feature
	NumResults         int      `json:"numResults,omitempty"`         // Number of results to return (default 10)
	IncludeDomains     []string `json:"includeDomains,omitempty"`     // Domains to include
	ExcludeDomains     []string `json:"excludeDomains,omitempty"`     // Domains to exclude
	StartCrawlDate     string   `json:"startCrawlDate,omitempty"`     // Start date for crawl filtering
	EndCrawlDate       string   `json:"endCrawlDate,omitempty"`       // End date for crawl filtering
	StartPublishedDate string   `json:"startPublishedDate,omitempty"` // Start date for published filtering
	EndPublishedDate   string   `json:"endPublishedDate,omitempty"`   // End date for published filtering
	Text               bool     `json:"text,omitempty"`               // Include text content
	Highlights         bool     `json:"highlights,omitempty"`         // Include highlights
	Summary            bool     `json:"summary,omitempty"`            // Include summary
}

// ExaSearchResult represents a single search result from Exa
type ExaSearchResult struct {
	Title         string                 `json:"title"`
	URL           string                 `json:"url"`
	ID            string                 `json:"id"`
	PublishedDate string                 `json:"publishedDate,omitempty"`
	Author        string                 `json:"author,omitempty"`
	Text          string                 `json:"text,omitempty"`
	Highlights    []string               `json:"highlights,omitempty"`
	Summary       string                 `json:"summary,omitempty"`
	Image         string                 `json:"image,omitempty"`
	Favicon       string                 `json:"favicon,omitempty"`
	Score         float64                `json:"score,omitempty"`
	Extras        map[string]interface{} `json:"extras,omitempty"`
}

// ExaSearchResponse represents the response from Exa search API
type ExaSearchResponse struct {
	RequestID          string                 `json:"requestId"`
	ResolvedSearchType string                 `json:"resolvedSearchType"`
	Results            []ExaSearchResult      `json:"results"`
	SearchType         string                 `json:"searchType"`
	Context            string                 `json:"context,omitempty"`
	CostDollars        map[string]interface{} `json:"costDollars,omitempty"`
}

// ExaSearchOptions provides options for customizing Exa searches
type ExaSearchOptions struct {
	NumResults        int      `json:"numResults,omitempty"`
	IncludeDomains    []string `json:"includeDomains,omitempty"`
	ExcludeDomains    []string `json:"excludeDomains,omitempty"`
	UseAutoprompt     bool     `json:"useAutoprompt,omitempty"`
	IncludeText       bool     `json:"includeText,omitempty"`
	IncludeHighlights bool     `json:"includeHighlights,omitempty"`
	IncludeSummary    bool     `json:"includeSummary,omitempty"`
	StartDate         string   `json:"startDate,omitempty"` // For both crawl and published date filtering
	EndDate           string   `json:"endDate,omitempty"`   // For both crawl and published date filtering
}

// Search performs a search using the Exa API
func (s *ExaSearchService) Search(ctx context.Context, query string, options *ExaSearchOptions) (*ExaSearchResponse, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("EXA_API_KEY environment variable not set")
	}

	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	// Set default options
	if options == nil {
		options = &ExaSearchOptions{}
	}

	// Set defaults
	numResults := options.NumResults
	if numResults == 0 {
		numResults = 10
	}

	// Build request payload
	reqPayload := ExaSearchRequest{
		Query:          query,
		Type:           "auto", // Let Exa choose the best search type
		NumResults:     numResults,
		UseAutoprompt:  options.UseAutoprompt,
		IncludeDomains: options.IncludeDomains,
		ExcludeDomains: options.ExcludeDomains,
		Text:           options.IncludeText,
		Highlights:     options.IncludeHighlights,
		Summary:        options.IncludeSummary,
	}

	// Add date filtering if provided
	if options.StartDate != "" {
		reqPayload.StartCrawlDate = options.StartDate
		reqPayload.StartPublishedDate = options.StartDate
	}
	if options.EndDate != "" {
		reqPayload.EndCrawlDate = options.EndDate
		reqPayload.EndPublishedDate = options.EndDate
	}

	// Marshal request to JSON
	jsonData, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.exa.ai/search", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.apiKey)

	// Make the request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Exa API returned status %d", resp.StatusCode)
	}

	// Parse response
	var searchResp ExaSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &searchResp, nil
}

// SearchWithDefaults performs a search with sensible defaults for content extraction
func (s *ExaSearchService) SearchWithDefaults(ctx context.Context, query string) (*ExaSearchResponse, error) {
	options := &ExaSearchOptions{
		NumResults:        5, // Limit to 5 results for efficiency
		IncludeText:       true,
		IncludeHighlights: true,
		IncludeSummary:    true,
		UseAutoprompt:     true, // Let Exa optimize the query
	}

	return s.Search(ctx, query, options)
}

// IsConfigured returns true if the service is properly configured
func (s *ExaSearchService) IsConfigured() bool {
	return s.apiKey != ""
}
