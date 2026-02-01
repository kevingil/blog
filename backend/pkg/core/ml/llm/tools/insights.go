package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"backend/pkg/api/dto"
	"backend/pkg/core/insight"
	"backend/pkg/core/ml"
	"backend/pkg/database"
	"backend/pkg/database/repository"

	"github.com/google/uuid"
)

// InsightService interface for insight operations
type InsightService interface {
	SearchInsightsByOrg(ctx context.Context, orgID uuid.UUID, query string, limit int) ([]dto.InsightResponse, error)
	SearchCrawledContentByOrg(ctx context.Context, orgID uuid.UUID, query string, limit int) ([]dto.CrawledContentResponse, error)
	GetTopicByID(ctx context.Context, id uuid.UUID) (*dto.InsightTopicResponse, error)
	ListTopics(ctx context.Context, orgID uuid.UUID) ([]dto.InsightTopicResponse, error)
	ListInsightsByTopic(ctx context.Context, topicID uuid.UUID, page, limit int) ([]dto.InsightResponse, int64, error)
}

// getInsightService creates and returns an insight service instance
func getInsightService() *insight.Service {
	db := database.DB()
	return insight.NewService(
		repository.NewInsightRepository(db),
		repository.NewInsightTopicRepository(db),
		repository.NewUserInsightStatusRepository(db),
		repository.NewCrawledContentRepository(db),
		repository.NewContentTopicMatchRepository(db),
		ml.NewEmbeddingService(),
	)
}

// organizationIDContextKey is the key for organization ID in context
type organizationIDContextKey string

const OrganizationIDContextKey organizationIDContextKey = "organization_id"

// WithOrganizationID adds organization ID to context for tools that need it
func WithOrganizationID(ctx context.Context, orgID uuid.UUID) context.Context {
	return context.WithValue(ctx, OrganizationIDContextKey, orgID)
}

// GetOrganizationIDFromContext retrieves the organization ID from context
func GetOrganizationIDFromContext(ctx context.Context) *uuid.UUID {
	orgID := ctx.Value(OrganizationIDContextKey)
	if orgID == nil {
		return nil
	}
	id := orgID.(uuid.UUID)
	return &id
}

// =============================================================================
// GetInsightsTool - Search for relevant insights
// =============================================================================

// GetInsightsTool allows the agent to search for relevant insights
type GetInsightsTool struct{}

// NewGetInsightsTool creates a new GetInsightsTool
func NewGetInsightsTool() *GetInsightsTool {
	return &GetInsightsTool{}
}

// Info returns the tool information
func (t *GetInsightsTool) Info() ToolInfo {
	return ToolInfo{
		Name: "get_insights",
		Description: `Search for relevant insights from your data pipeline.

Insights are AI-generated summaries of content from your preferred data sources (blogs, forums, etc.).
Use this tool to find relevant context and information when writing articles.

WHEN TO USE:
- When you need background information on a topic
- When looking for recent trends or news in a specific area
- When you want to reference or cite insights in the article
- Before writing content to understand what's happening in the space

RETURNS:
A list of relevant insights with title, summary, key points, and topic information.`,
		Parameters: map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search query to find relevant insights (e.g., 'AI trends 2024', 'React performance optimization')",
			},
			"limit": map[string]any{
				"type":        "number",
				"description": "Maximum number of insights to return (default: 5, max: 15)",
			},
		},
		Required: []string{"query"},
	}
}

// Run executes the tool
func (t *GetInsightsTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
	var input struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return NewTextErrorResponse("Failed to parse input parameters"), nil
	}

	if input.Query == "" {
		return NewTextErrorResponse("Query is required"), nil
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 5
	}
	if limit > 15 {
		limit = 15
	}

	log.Printf("🔍 [GetInsights] Searching for: %s (limit: %d)", input.Query, limit)

	orgID := GetOrganizationIDFromContext(ctx)
	if orgID == nil {
		return NewTextErrorResponse("Organization context not available"), nil
	}

	svc := getInsightService()
	insights, err := svc.SearchInsightsByOrg(ctx, *orgID, input.Query, limit)
	if err != nil {
		log.Printf("🔍 [GetInsights] Error: %v", err)
		return NewTextErrorResponse(fmt.Sprintf("Failed to search insights: %v", err)), nil
	}

	if len(insights) == 0 {
		return NewTextResponse("No relevant insights found for your query. Try a different search term or check if you have data sources configured."), nil
	}

	// Format response
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d relevant insights:\n\n", len(insights)))

	for i, ins := range insights {
		sb.WriteString(fmt.Sprintf("## %d. %s\n", i+1, ins.Title))
		sb.WriteString(fmt.Sprintf("**Summary:** %s\n", ins.Summary))

		if len(ins.KeyPoints) > 0 {
			sb.WriteString("\n**Key Points:**\n")
			for _, point := range ins.KeyPoints {
				sb.WriteString(fmt.Sprintf("- %s\n", point))
			}
		}

		if ins.TopicName != nil && *ins.TopicName != "" {
			sb.WriteString(fmt.Sprintf("\n*Topic: %s*\n", *ins.TopicName))
		}

		sb.WriteString(fmt.Sprintf("*Generated: %s*\n\n", ins.GeneratedAt.Format("2006-01-02")))
		sb.WriteString("---\n\n")
	}

	// Create result for UI
	result := map[string]interface{}{
		"insights": insights,
		"count":    len(insights),
		"query":    input.Query,
	}

	return NewTextResponseWithResult(sb.String(), result), nil
}

// Parallelizable indicates this tool can run in parallel
func (t *GetInsightsTool) Parallelizable() bool {
	return true
}

// =============================================================================
// SearchCrawledContentTool - Search crawled content from data sources
// =============================================================================

// SearchCrawledContentTool allows the agent to search crawled content
type SearchCrawledContentTool struct{}

// NewSearchCrawledContentTool creates a new SearchCrawledContentTool
func NewSearchCrawledContentTool() *SearchCrawledContentTool {
	return &SearchCrawledContentTool{}
}

// Info returns the tool information
func (t *SearchCrawledContentTool) Info() ToolInfo {
	return ToolInfo{
		Name: "search_crawled_content",
		Description: `Search through raw crawled content from your data sources.

This searches the actual articles and posts that have been crawled from your preferred websites.
Use this for more detailed research when insights don't provide enough context.

WHEN TO USE:
- When you need detailed information from original sources
- When looking for specific quotes or data
- When you want to reference the original article content
- For deeper research beyond summarized insights

RETURNS:
A list of crawled content items with title, URL, summary, and excerpt.`,
		Parameters: map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search query to find relevant content (e.g., 'machine learning deployment', 'TypeScript best practices')",
			},
			"limit": map[string]any{
				"type":        "number",
				"description": "Maximum number of content items to return (default: 5, max: 10)",
			},
		},
		Required: []string{"query"},
	}
}

// Run executes the tool
func (t *SearchCrawledContentTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
	var input struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return NewTextErrorResponse("Failed to parse input parameters"), nil
	}

	if input.Query == "" {
		return NewTextErrorResponse("Query is required"), nil
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 5
	}
	if limit > 10 {
		limit = 10
	}

	log.Printf("📚 [SearchCrawledContent] Searching for: %s (limit: %d)", input.Query, limit)

	orgID := GetOrganizationIDFromContext(ctx)
	if orgID == nil {
		return NewTextErrorResponse("Organization context not available"), nil
	}

	svc := getInsightService()
	contents, err := svc.SearchCrawledContentByOrg(ctx, *orgID, input.Query, limit)
	if err != nil {
		log.Printf("📚 [SearchCrawledContent] Error: %v", err)
		return NewTextErrorResponse(fmt.Sprintf("Failed to search content: %v", err)), nil
	}

	if len(contents) == 0 {
		return NewTextResponse("No relevant content found for your query. Try a different search term or check if you have data sources configured and crawled."), nil
	}

	// Format response
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d relevant content items:\n\n", len(contents)))

	for i, c := range contents {
		title := "Untitled"
		if c.Title != nil {
			title = *c.Title
		}

		sb.WriteString(fmt.Sprintf("## %d. %s\n", i+1, title))
		sb.WriteString(fmt.Sprintf("**URL:** %s\n", c.URL))

		if c.Summary != nil && *c.Summary != "" {
			sb.WriteString(fmt.Sprintf("\n**Summary:** %s\n", *c.Summary))
		}

		// Include excerpt of content (first 500 chars)
		excerpt := c.Content
		if len(excerpt) > 500 {
			excerpt = excerpt[:500] + "..."
		}
		sb.WriteString(fmt.Sprintf("\n**Excerpt:**\n%s\n", excerpt))

		if c.Author != nil && *c.Author != "" {
			sb.WriteString(fmt.Sprintf("\n*Author: %s*\n", *c.Author))
		}

		if c.PublishedAt != nil {
			sb.WriteString(fmt.Sprintf("*Published: %s*\n", c.PublishedAt.Format("2006-01-02")))
		}

		sb.WriteString("\n---\n\n")
	}

	// Create result for UI
	result := map[string]interface{}{
		"contents": contents,
		"count":    len(contents),
		"query":    input.Query,
	}

	return NewTextResponseWithResult(sb.String(), result), nil
}

// Parallelizable indicates this tool can run in parallel
func (t *SearchCrawledContentTool) Parallelizable() bool {
	return true
}

// =============================================================================
// GetTopicSummaryTool - Get the latest insight for a topic
// =============================================================================

// GetTopicSummaryTool allows the agent to get summary for a specific topic
type GetTopicSummaryTool struct{}

// NewGetTopicSummaryTool creates a new GetTopicSummaryTool
func NewGetTopicSummaryTool() *GetTopicSummaryTool {
	return &GetTopicSummaryTool{}
}

// Info returns the tool information
func (t *GetTopicSummaryTool) Info() ToolInfo {
	return ToolInfo{
		Name: "get_topic_summary",
		Description: `Get the latest insights for a specific topic.

Topics are categories that organize your crawled content and insights.
Use this to get focused information on a particular area of interest.

WHEN TO USE:
- When you want insights specifically about a known topic
- When browsing what topics are available
- To get the latest summary for a topic area

If topic_id is not provided, returns a list of available topics.`,
		Parameters: map[string]any{
			"topic_id": map[string]any{
				"type":        "string",
				"description": "UUID of the topic to get insights for. If empty, lists available topics.",
			},
		},
		Required: []string{},
	}
}

// Run executes the tool
func (t *GetTopicSummaryTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
	var input struct {
		TopicID string `json:"topic_id"`
	}
	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		// Ignore parse errors for optional params
	}

	orgID := GetOrganizationIDFromContext(ctx)
	if orgID == nil {
		return NewTextErrorResponse("Organization context not available"), nil
	}

	svc := getInsightService()

	// If no topic ID provided, list available topics
	if input.TopicID == "" {
		log.Printf("📋 [GetTopicSummary] Listing available topics")

		topics, err := svc.ListTopics(ctx, *orgID)
		if err != nil {
			return NewTextErrorResponse(fmt.Sprintf("Failed to list topics: %v", err)), nil
		}

		if len(topics) == 0 {
			return NewTextResponse("No topics configured yet. Topics are created to categorize your data sources and insights."), nil
		}

		var sb strings.Builder
		sb.WriteString("Available topics:\n\n")

		for _, topic := range topics {
			sb.WriteString(fmt.Sprintf("- **%s** (ID: %s)\n", topic.Name, topic.ID))
			if topic.Description != nil && *topic.Description != "" {
				sb.WriteString(fmt.Sprintf("  %s\n", *topic.Description))
			}
			sb.WriteString(fmt.Sprintf("  Content count: %d\n", topic.ContentCount))
			sb.WriteString("\n")
		}

		result := map[string]interface{}{
			"topics": topics,
			"count":  len(topics),
		}

		return NewTextResponseWithResult(sb.String(), result), nil
	}

	// Get insights for specific topic
	topicID, err := uuid.Parse(input.TopicID)
	if err != nil {
		return NewTextErrorResponse("Invalid topic ID format"), nil
	}

	log.Printf("📋 [GetTopicSummary] Getting insights for topic: %s", topicID)

	// Get topic details
	topic, err := svc.GetTopicByID(ctx, topicID)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("Failed to get topic: %v", err)), nil
	}

	// Get latest insights for topic
	insights, total, err := svc.ListInsightsByTopic(ctx, topicID, 1, 5)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("Failed to get insights: %v", err)), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Topic: %s\n\n", topic.Name))

	if topic.Description != nil && *topic.Description != "" {
		sb.WriteString(fmt.Sprintf("*%s*\n\n", *topic.Description))
	}

	sb.WriteString(fmt.Sprintf("Total content items: %d\n", topic.ContentCount))
	sb.WriteString(fmt.Sprintf("Total insights: %d\n\n", total))

	if len(insights) == 0 {
		sb.WriteString("No insights generated for this topic yet.\n")
	} else {
		sb.WriteString("## Latest Insights\n\n")
		for i, ins := range insights {
			sb.WriteString(fmt.Sprintf("### %d. %s\n", i+1, ins.Title))
			sb.WriteString(fmt.Sprintf("%s\n", ins.Summary))

			if len(ins.KeyPoints) > 0 {
				sb.WriteString("\n**Key Points:**\n")
				for _, point := range ins.KeyPoints {
					sb.WriteString(fmt.Sprintf("- %s\n", point))
				}
			}
			sb.WriteString("\n")
		}
	}

	result := map[string]interface{}{
		"topic":    topic,
		"insights": insights,
		"total":    total,
	}

	return NewTextResponseWithResult(sb.String(), result), nil
}

// Parallelizable indicates this tool can run in parallel
func (t *GetTopicSummaryTool) Parallelizable() bool {
	return true
}
