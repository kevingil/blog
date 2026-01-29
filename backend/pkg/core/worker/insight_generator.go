package worker

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"backend/pkg/core/insight"
	"backend/pkg/database"
	"backend/pkg/database/repository"
	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// InsightWorker generates insights from crawled content
type InsightWorker struct {
	logger           *slog.Logger
	openaiClient     *openai.Client
	minContentCount  int
	maxContentPerGen int
}

// NewInsightWorker creates a new InsightWorker instance
func NewInsightWorker(logger *slog.Logger, openaiAPIKey string) *InsightWorker {
	if logger == nil {
		logger = slog.Default()
	}

	var client *openai.Client
	if openaiAPIKey != "" {
		c := openai.NewClient(option.WithAPIKey(openaiAPIKey))
		client = &c
	}

	return &InsightWorker{
		logger:           logger,
		openaiClient:     client,
		minContentCount:  3,  // Minimum content items to generate an insight
		maxContentPerGen: 10, // Maximum content items per insight
	}
}

// Name returns the worker name
func (w *InsightWorker) Name() string {
	return "insight_worker"
}

// Run executes the insight worker
func (w *InsightWorker) Run(ctx context.Context) error {
	w.logger.Info("starting insight worker run")

	if w.openaiClient == nil {
		w.logger.Warn("OpenAI client not configured, skipping insight generation")
		return nil
	}

	// Get all topics
	topicRepo := repository.NewInsightTopicRepository(database.DB())
	topics, err := topicRepo.FindAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to get topics: %w", err)
	}

	if len(topics) == 0 {
		w.logger.Info("no topics found, skipping insight generation")
		return nil
	}

	w.logger.Info("processing topics for insights", "count", len(topics))

	for _, topic := range topics {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := w.generateInsightForTopic(ctx, &topic); err != nil {
				w.logger.Error("failed to generate insight for topic", "topic_id", topic.ID, "topic_name", topic.Name, "error", err)
			}
		}
	}

	return nil
}

// generateInsightForTopic generates an insight for a specific topic
func (w *InsightWorker) generateInsightForTopic(ctx context.Context, topic *types.InsightTopic) error {
	matchRepo := repository.NewContentTopicMatchRepository(database.DB())
	contentRepo := repository.NewCrawledContentRepository(database.DB())

	// Get recent content matches for this topic
	matches, total, err := matchRepo.FindPrimaryByTopicID(ctx, topic.ID, 0, w.maxContentPerGen)
	if err != nil {
		return fmt.Errorf("failed to get content matches: %w", err)
	}

	if total < int64(w.minContentCount) {
		w.logger.Debug("not enough content for insight generation", "topic_id", topic.ID, "content_count", total)
		return nil
	}

	// Get content details
	contentIDs := make([]uuid.UUID, len(matches))
	for i, m := range matches {
		contentIDs[i] = m.ContentID
	}

	contents, err := contentRepo.FindByIDs(ctx, contentIDs)
	if err != nil {
		return fmt.Errorf("failed to get content details: %w", err)
	}

	if len(contents) < w.minContentCount {
		return nil
	}

	// Check if we should generate a new insight
	// (e.g., haven't generated one recently or have new content)
	if topic.LastInsightAt != nil {
		// Only generate if last insight was more than 24 hours ago
		if time.Since(*topic.LastInsightAt) < 24*time.Hour {
			w.logger.Debug("recent insight exists, skipping", "topic_id", topic.ID)
			return nil
		}
	}

	// Build content summary for LLM
	contentSummary := w.buildContentSummary(contents)

	// Generate insight using LLM
	insightData, err := w.generateInsightWithLLM(ctx, topic, contentSummary)
	if err != nil {
		return fmt.Errorf("failed to generate insight with LLM: %w", err)
	}

	// Determine period
	var periodStart, periodEnd *time.Time
	for _, c := range contents {
		if c.PublishedAt != nil {
			if periodStart == nil || c.PublishedAt.Before(*periodStart) {
				periodStart = c.PublishedAt
			}
			if periodEnd == nil || c.PublishedAt.After(*periodEnd) {
				periodEnd = c.PublishedAt
			}
		}
	}
	if periodStart == nil {
		now := time.Now()
		weekAgo := now.Add(-7 * 24 * time.Hour)
		periodStart = &weekAgo
		periodEnd = &now
	}

	// Create the insight
	_, err = insight.CreateInsight(
		ctx,
		topic.OrganizationID,
		&topic.ID,
		insightData.Title,
		insightData.Summary,
		insightData.Content,
		insightData.KeyPoints,
		contentIDs,
		periodStart,
		periodEnd,
	)
	if err != nil {
		return fmt.Errorf("failed to create insight: %w", err)
	}

	w.logger.Info("generated insight for topic", "topic_id", topic.ID, "topic_name", topic.Name, "title", insightData.Title)
	return nil
}

// buildContentSummary builds a summary of content for LLM processing
func (w *InsightWorker) buildContentSummary(contents []types.CrawledContent) string {
	var sb strings.Builder

	for i, c := range contents {
		sb.WriteString(fmt.Sprintf("## Article %d\n", i+1))
		if c.Title != nil {
			sb.WriteString(fmt.Sprintf("Title: %s\n", *c.Title))
		}
		sb.WriteString(fmt.Sprintf("URL: %s\n", c.URL))
		if c.PublishedAt != nil {
			sb.WriteString(fmt.Sprintf("Published: %s\n", c.PublishedAt.Format("2006-01-02")))
		}

		// Truncate content to ~1500 chars per article
		content := c.Content
		if len(content) > 1500 {
			content = content[:1500] + "..."
		}
		sb.WriteString(fmt.Sprintf("\nContent:\n%s\n\n", content))
		sb.WriteString("---\n\n")
	}

	return sb.String()
}

// insightLLMResponse represents the structured response from LLM
type insightLLMResponse struct {
	Title     string
	Summary   string
	Content   string
	KeyPoints []string
}

// generateInsightWithLLM generates insight content using OpenAI
func (w *InsightWorker) generateInsightWithLLM(ctx context.Context, topic *types.InsightTopic, contentSummary string) (*insightLLMResponse, error) {
	systemPrompt := `You are an expert content analyst and writer. Your task is to analyze multiple articles on a specific topic and generate a comprehensive insight summary.

Your output should include:
1. A compelling title for the insight
2. A 2-3 sentence summary
3. A full "mini blog" content (2-4 paragraphs) that synthesizes the key information
4. 3-5 key takeaways as bullet points

Format your response exactly as follows:
TITLE: [Your title here]
SUMMARY: [Your 2-3 sentence summary]
CONTENT: [Your full mini blog content]
KEY_POINTS:
- [Point 1]
- [Point 2]
- [Point 3]`

	userPrompt := fmt.Sprintf(`Topic: %s
Description: %s

Please analyze the following articles and generate an insight:

%s`, topic.Name, stringValue(topic.Description), contentSummary)

	resp, err := w.openaiClient.Chat.Completions.New(
		ctx,
		openai.ChatCompletionNewParams{
			Model: openai.ChatModelGPT4oMini,
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage(systemPrompt),
				openai.UserMessage(userPrompt),
			},
			MaxCompletionTokens: openai.Int(2000),
			Temperature:         openai.Float(0.7),
		},
	)

	if err != nil {
		return nil, fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	// Parse the response
	return w.parseInsightResponse(resp.Choices[0].Message.Content)
}

// parseInsightResponse parses the LLM response into structured data
func (w *InsightWorker) parseInsightResponse(response string) (*insightLLMResponse, error) {
	result := &insightLLMResponse{}

	// Extract title
	if idx := strings.Index(response, "TITLE:"); idx != -1 {
		endIdx := strings.Index(response[idx:], "\n")
		if endIdx == -1 {
			endIdx = len(response) - idx
		}
		result.Title = strings.TrimSpace(response[idx+6 : idx+endIdx])
	}

	// Extract summary
	if idx := strings.Index(response, "SUMMARY:"); idx != -1 {
		// Find end (either CONTENT: or KEY_POINTS:)
		endIdx := strings.Index(response[idx:], "CONTENT:")
		if endIdx == -1 {
			endIdx = strings.Index(response[idx:], "KEY_POINTS:")
		}
		if endIdx == -1 {
			endIdx = len(response) - idx
		}
		result.Summary = strings.TrimSpace(response[idx+8 : idx+endIdx])
	}

	// Extract content
	if idx := strings.Index(response, "CONTENT:"); idx != -1 {
		endIdx := strings.Index(response[idx:], "KEY_POINTS:")
		if endIdx == -1 {
			endIdx = len(response) - idx
		}
		result.Content = strings.TrimSpace(response[idx+8 : idx+endIdx])
	}

	// Extract key points
	if idx := strings.Index(response, "KEY_POINTS:"); idx != -1 {
		keyPointsSection := response[idx+11:]
		lines := strings.Split(keyPointsSection, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "•") {
				point := strings.TrimPrefix(strings.TrimPrefix(line, "-"), "•")
				point = strings.TrimSpace(point)
				if point != "" {
					result.KeyPoints = append(result.KeyPoints, point)
				}
			}
		}
	}

	// Validate required fields
	if result.Title == "" {
		result.Title = "Insight Summary"
	}
	if result.Summary == "" {
		return nil, fmt.Errorf("failed to extract summary from LLM response")
	}

	return result, nil
}

func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
