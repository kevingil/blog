// Package exa provides a client for the Exa search API
package exa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Client handles searching the web using Exa API
type Client struct {
	apiKey string
	client *http.Client
}

// NewClient creates a new Exa client
func NewClient() *Client {
	apiKey := os.Getenv("EXA_API_KEY")
	if apiKey == "" {
		// This is not fatal - the tool will handle missing API key gracefully
		return &Client{
			client: &http.Client{Timeout: 30 * time.Second},
		}
	}

	return &Client{
		apiKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// SearchRequest represents the request payload for Exa search
type SearchRequest struct {
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

// SearchResult represents a single search result from Exa
type SearchResult struct {
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

// SearchResponse represents the response from Exa search API
type SearchResponse struct {
	RequestID          string                 `json:"requestId"`
	ResolvedSearchType string                 `json:"resolvedSearchType"`
	Results            []SearchResult         `json:"results"`
	SearchType         string                 `json:"searchType"`
	Context            string                 `json:"context,omitempty"`
	CostDollars        map[string]interface{} `json:"costDollars,omitempty"`
}

// SearchOptions provides options for customizing Exa searches
type SearchOptions struct {
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
func (c *Client) Search(ctx context.Context, query string, options *SearchOptions) (*SearchResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("EXA_API_KEY environment variable not set")
	}

	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	// Set default options
	if options == nil {
		options = &SearchOptions{}
	}

	// Set defaults
	numResults := options.NumResults
	if numResults == 0 {
		numResults = 10
	}

	// Build request payload
	reqPayload := SearchRequest{
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
	req.Header.Set("x-api-key", c.apiKey)

	// Make the request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Exa API returned status %d", resp.StatusCode)
	}

	// Parse response
	var searchResp SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &searchResp, nil
}

// SearchWithDefaults performs a search with sensible defaults for content extraction
func (c *Client) SearchWithDefaults(ctx context.Context, query string) (*SearchResponse, error) {
	options := &SearchOptions{
		NumResults:        6, // Always return 6 results for comprehensive coverage
		IncludeText:       true,
		IncludeHighlights: true,
		IncludeSummary:    true,
		UseAutoprompt:     true, // Let Exa optimize the query
	}

	return c.Search(ctx, query, options)
}

// IsConfigured returns true if the client is properly configured
func (c *Client) IsConfigured() bool {
	return c.apiKey != ""
}

// AnswerRequest represents the request payload for Exa answer endpoint
type AnswerRequest struct {
	Query string `json:"query"`
	Text  bool   `json:"text,omitempty"` // Include full text content in citations
}

// Citation represents a citation from the Exa answer API
type Citation struct {
	ID            string                 `json:"id"`
	URL           string                 `json:"url"`
	Title         string                 `json:"title"`
	Author        string                 `json:"author,omitempty"`
	PublishedDate string                 `json:"publishedDate,omitempty"`
	Text          string                 `json:"text,omitempty"`
	Image         string                 `json:"image,omitempty"`
	Favicon       string                 `json:"favicon,omitempty"`
	Extras        map[string]interface{} `json:"extras,omitempty"`
}

// AnswerResponse represents the response from Exa answer API
type AnswerResponse struct {
	Answer      string                 `json:"answer"`
	Citations   []Citation             `json:"citations"`
	CostDollars map[string]interface{} `json:"costDollars,omitempty"`
}

// Answer gets a direct answer to a question using the Exa /answer endpoint
func (c *Client) Answer(ctx context.Context, question string, includeText bool) (*AnswerResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("EXA_API_KEY environment variable not set")
	}

	if question == "" {
		return nil, fmt.Errorf("question cannot be empty")
	}

	// Build request payload
	reqPayload := AnswerRequest{
		Query: question,
		Text:  includeText,
	}

	// Marshal request to JSON
	jsonData, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.exa.ai/answer", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)

	// Make the request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Exa Answer API returned status %d", resp.StatusCode)
	}

	// Parse response
	var answerResp AnswerResponse
	if err := json.NewDecoder(resp.Body).Decode(&answerResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &answerResp, nil
}

// AnswerWithDefaults gets an answer with sensible defaults
func (c *Client) AnswerWithDefaults(ctx context.Context, question string) (*AnswerResponse, error) {
	return c.Answer(ctx, question, true) // Include full text by default
}
