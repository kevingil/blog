package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"backend/pkg/core/source"
	"backend/pkg/integrations/exa"
	"backend/pkg/types"

	"github.com/google/uuid"
)

// ExaSearchService interface for Exa search operations
// Satisfied directly by exa.Client
type ExaSearchService interface {
	SearchWithDefaults(ctx context.Context, query string) (*exa.SearchResponse, error)
	IsConfigured() bool
}

// SourceCreator interface for creating sources from search results
// Satisfied directly by source.Service
type SourceCreator interface {
	Create(ctx context.Context, req source.CreateRequest) (*types.Source, error)
}

// ExaSearchTool searches the web using Exa and automatically creates sources
type ExaSearchTool struct {
	exaService    ExaSearchService
	sourceCreator SourceCreator
}

func NewExaSearchTool(exaService ExaSearchService, sourceCreator SourceCreator) *ExaSearchTool {
	return &ExaSearchTool{
		exaService:    exaService,
		sourceCreator: sourceCreator,
	}
}

func (t *ExaSearchTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "search_web_sources",
		Description: "Search the web for sources on a topic. Creates citable sources automatically. Be specific in queries.",
		Parameters: map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "A specific search query. Include technology names, timeframes, or specific aspects.",
			},
			"create_sources": map[string]any{
				"type":        []string{"boolean", "null"},
				"description": "Whether to automatically create sources from the search results (default: true)",
			},
		},
		Required: []string{"query"},
	}
}

func (t *ExaSearchTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
	var input struct {
		Query         string `json:"query"`
		CreateSources *bool  `json:"create_sources"`
	}

	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		log.Printf("🔍 [ExaSearch] ERROR: Failed to parse input: %v", err)
		return NewTextErrorResponse("Invalid input format"), err
	}

	if input.Query == "" {
		log.Printf("🔍 [ExaSearch] ERROR: Empty query provided")
		return NewTextErrorResponse("query is required"), fmt.Errorf("query is required")
	}

	// Default to creating sources unless explicitly set to false
	createSources := true
	if input.CreateSources != nil {
		createSources = *input.CreateSources
	}

	log.Printf("🔍 [ExaSearch] Starting web search with Exa")
	log.Printf("   📝 Query: %q", input.Query)
	log.Printf("   📂 Create sources: %t", createSources)

	// Check if Exa service is configured
	if !t.exaService.IsConfigured() {
		log.Printf("🔍 [ExaSearch] ERROR: Exa service not configured (missing EXA_API_KEY)")
		return NewTextErrorResponse("Exa search service not configured. Please set EXA_API_KEY environment variable."), fmt.Errorf("exa service not configured")
	}

	// Get article ID from context if creating sources
	var articleID uuid.UUID
	var err error
	if createSources {
		articleIDStr := GetArticleIDFromContext(ctx)
		if articleIDStr == "" {
			log.Printf("🔍 [ExaSearch] WARNING: No article ID in context - will return search results but cannot create sources")
			createSources = false
		} else {
			articleID, err = uuid.Parse(articleIDStr)
			if err != nil {
				log.Printf("🔍 [ExaSearch] ERROR: Invalid article ID format: %s", articleIDStr)
				return NewTextErrorResponse("Invalid article ID"), fmt.Errorf("invalid article ID: %w", err)
			}
		}
	}

	// Perform Exa search
	log.Printf("🔍 [ExaSearch] Executing Exa web search...")
	searchResp, err := t.exaService.SearchWithDefaults(ctx, input.Query)
	if err != nil {
		log.Printf("🔍 [ExaSearch] ERROR: Exa search failed: %v", err)
		return NewTextErrorResponse(fmt.Sprintf("Failed to search web: %v", err)), err
	}

	if len(searchResp.Results) == 0 {
		log.Printf("🔍 [ExaSearch] No results found for query: %q", input.Query)
		result := map[string]interface{}{
			"search_results":     []map[string]interface{}{},
			"sources_created":    []map[string]interface{}{},
			"query":              input.Query,
			"total_found":        0,
			"sources_attempted":  0,
			"sources_successful": 0,
			"tool_name":          "search_web_sources",
			"message":            "No results found for the search query",
		}
		resultJSON, _ := json.Marshal(result)
		return NewTextResponse(string(resultJSON)), nil
	}

	log.Printf("🔍 [ExaSearch] ✅ Found %d search results", len(searchResp.Results))

	// Limit results to max_results
	results := searchResp.Results

	// Process search results
	var searchResults []map[string]interface{}
	var sourcesCreated []map[string]interface{}
	var sourcesAttempted int
	var sourcesSuccessful int

	for i, result := range results {
		log.Printf("🔍 [ExaSearch] Processing result #%d:", i+1)
		log.Printf("   📋 Title: %q", result.Title)
		log.Printf("   🔗 URL: %q", result.URL)
		log.Printf("   📅 Published: %q", result.PublishedDate)
		if result.Author != "" {
			log.Printf("   👤 Author: %q", result.Author)
		}
		if result.Text != "" {
			log.Printf("   📄 Full text length: %d characters", len(result.Text))
		}

		// Create search result entry
		searchResult := map[string]interface{}{
			"title":          result.Title,
			"url":            result.URL,
			"id":             result.ID,
			"published_date": result.PublishedDate,
			"author":         result.Author,
			"summary":        result.Summary,
		}

		// Add highlights if available
		if len(result.Highlights) > 0 {
			searchResult["highlights"] = result.Highlights
		}

		// Add text content if available (truncated for readability)
		if result.Text != "" {
			textPreview := result.Text
			if len(textPreview) > 500 {
				textPreview = textPreview[:500] + "..."
			}
			searchResult["text_preview"] = textPreview
			searchResult["text_length"] = len(result.Text)
			searchResult["has_full_text"] = true
		} else {
			searchResult["has_full_text"] = false
		}

		searchResults = append(searchResults, searchResult)

		// Attempt to create source if enabled and we have full text content
		if createSources && t.sourceCreator != nil && result.Text != "" {
			sourcesAttempted++
			log.Printf("🔍 [ExaSearch] Creating web content source from search result: %s", result.URL)

			// Skip obviously problematic URLs
			if t.shouldSkipURL(result.URL) {
				log.Printf("🔍 [ExaSearch] ⚠️  Skipping URL (unsupported type): %s", result.URL)
				continue
			}

			// Create WebContentSource from search result data
			searchResultData := SearchResultData{
				ID:            result.ID,
				Title:         result.Title,
				URL:           result.URL,
				Text:          result.Text,
				Summary:       result.Summary,
				Author:        result.Author,
				PublishedDate: result.PublishedDate,
				Highlights:    result.Highlights,
				Score:         result.Score,
				Favicon:       result.Favicon,
				Image:         result.Image,
				Extras:        result.Extras,
			}

			webContentSource := NewWebContentSourceFromSearchResult(articleID, input.Query, searchResultData)

			// Convert to ArticleSource and create in database
			articleSource := webContentSource.ToSource()

			// Use the source service directly to create the source with embedding generation
			createdSource, err := t.sourceCreator.Create(ctx, source.CreateRequest{
				ArticleID:  articleSource.ArticleID,
				Title:      articleSource.Title,
				Content:    articleSource.Content,
				URL:        articleSource.URL,
				SourceType: "web_search", // Specify this as a web search source
			})
			if err != nil {
				log.Printf("🔍 [ExaSearch] ❌ Failed to create web content source from %s: %v", result.URL, err)
				// Continue with other results even if one fails
				continue
			}

			sourcesSuccessful++
			log.Printf("🔍 [ExaSearch] ✅ Successfully created web content source from %s (ID: %s)", result.URL, createdSource.ID)

			sourceInfo := map[string]interface{}{
				"original_title":   result.Title,
				"original_url":     result.URL,
				"source_created":   true,
				"search_result_id": result.ID,
				"source_id":        createdSource.ID,
				"content_length":   len(result.Text),
				"source_type":      "web_search",
				"search_query":     input.Query,
			}

			sourcesCreated = append(sourcesCreated, sourceInfo)
		}
	}

	// Prepare response
	result := map[string]interface{}{
		"search_results":     searchResults,
		"sources_created":    sourcesCreated,
		"query":              input.Query,
		"total_found":        len(searchResp.Results),
		"results_processed":  len(results),
		"sources_attempted":  sourcesAttempted,
		"sources_successful": sourcesSuccessful,
		"tool_name":          "search_web_sources",
		"exa_request_id":     searchResp.RequestID,
		"search_type":        searchResp.ResolvedSearchType,
	}

	// Add cost information if available
	if searchResp.CostDollars != nil {
		result["cost_info"] = searchResp.CostDollars
	}

	// Create summary message
	message := fmt.Sprintf("Found %d search results", len(searchResults))
	if createSources {
		if sourcesSuccessful > 0 {
			message += fmt.Sprintf(" and successfully created %d sources", sourcesSuccessful)
		} else if sourcesAttempted > 0 {
			message += " but failed to create any sources"
		} else {
			message += " but source creation was skipped"
		}
	}
	result["message"] = message

	// Log summary
	log.Printf("🔍 [ExaSearch] 📊 Summary:")
	log.Printf("   🔍 Search results: %d (from %d total found)", len(results), len(searchResp.Results))
	if createSources {
		log.Printf("   📂 Sources attempted: %d", sourcesAttempted)
		log.Printf("   ✅ Sources created: %d", sourcesSuccessful)
		if sourcesAttempted > 0 {
			successRate := float64(sourcesSuccessful) / float64(sourcesAttempted) * 100
			log.Printf("   📊 Success rate: %.1f%%", successRate)
		}
	}
	log.Printf("   🔎 Exa search type: %s", searchResp.ResolvedSearchType)

	log.Printf("🔍 [ExaSearch] ✅ Web search completed successfully")

	// Create artifact hint for sources display
	artifactData := map[string]interface{}{
		"search_results":     searchResults,
		"sources_created":    sourcesCreated,
		"query":              input.Query,
		"total_found":        len(searchResp.Results),
		"sources_successful": sourcesSuccessful,
	}

	resultJSON, _ := json.Marshal(result)
	return ToolResponse{
		Type:    ToolResponseTypeText,
		Content: string(resultJSON),
		Result:  result,
		Artifact: &ArtifactHint{
			Type: ArtifactHintTypeSources,
			Data: artifactData,
		},
	}, nil
}

// shouldSkipURL determines if a URL should be skipped for source creation
func (t *ExaSearchTool) shouldSkipURL(url string) bool {
	url = strings.ToLower(url)

	// Skip social media URLs that are unlikely to have good content
	skipDomains := []string{
		"twitter.com",
		"x.com",
		"facebook.com",
		"instagram.com",
		"linkedin.com/feed",
		"tiktok.com",
		"youtube.com/shorts",
		"reddit.com/r/", // Skip specific subreddit posts, but allow reddit.com articles
	}

	for _, domain := range skipDomains {
		if strings.Contains(url, domain) {
			return true
		}
	}

	// Skip URLs that are likely to be problematic
	skipPatterns := []string{
		"/search?",
		"/login",
		"/signup",
		"/register",
		"javascript:",
		"mailto:",
		"tel:",
	}

	for _, pattern := range skipPatterns {
		if strings.Contains(url, pattern) {
			return true
		}
	}

	return false
}
