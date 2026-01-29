package insight

import (
	"context"
	"fmt"
	"strings"
	"time"

	"backend/pkg/core/ml"
	"backend/pkg/database"
	"backend/pkg/database/repository"
	"backend/pkg/types"

	"github.com/google/uuid"
)

// getInsightRepo returns an insight repository instance
func getInsightRepo() *repository.InsightRepository {
	return repository.NewInsightRepository(database.DB())
}

// getTopicRepo returns an insight topic repository instance
func getTopicRepo() *repository.InsightTopicRepository {
	return repository.NewInsightTopicRepository(database.DB())
}

// getCrawledContentRepo returns a crawled content repository instance
func getCrawledContentRepo() *repository.CrawledContentRepository {
	return repository.NewCrawledContentRepository(database.DB())
}

// getContentTopicMatchRepo returns a content topic match repository instance
func getContentTopicMatchRepo() *repository.ContentTopicMatchRepository {
	return repository.NewContentTopicMatchRepository(database.DB())
}

// getEmbeddingService returns an embedding service instance
func getEmbeddingService() *ml.EmbeddingService {
	return ml.NewEmbeddingService()
}

// =============================================================================
// Insight Functions
// =============================================================================

// GetInsightByID retrieves an insight by its ID
func GetInsightByID(ctx context.Context, id uuid.UUID) (*types.InsightResponse, error) {
	repo := getInsightRepo()
	insight, err := repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toInsightResponse(insight), nil
}

// GetInsightWithSources retrieves an insight with its source content
func GetInsightWithSources(ctx context.Context, id uuid.UUID) (*types.InsightWithSources, error) {
	insightRepo := getInsightRepo()
	contentRepo := getCrawledContentRepo()
	topicRepo := getTopicRepo()

	insight, err := insightRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Get source contents
	var sourceContents []types.CrawledContentResponse
	if len(insight.SourceContentIDs) > 0 {
		contents, err := contentRepo.FindByIDs(ctx, insight.SourceContentIDs)
		if err != nil {
			return nil, err
		}
		sourceContents = make([]types.CrawledContentResponse, len(contents))
		for i, c := range contents {
			sourceContents[i] = types.CrawledContentResponse{
				ID:           c.ID,
				DataSourceID: c.DataSourceID,
				URL:          c.URL,
				Title:        c.Title,
				Content:      c.Content,
				Summary:      c.Summary,
				Author:       c.Author,
				PublishedAt:  c.PublishedAt,
				MetaData:     c.MetaData,
				CreatedAt:    c.CreatedAt,
			}
		}
	}

	// Get topic if present
	var topicResponse *types.InsightTopicResponse
	if insight.TopicID != nil {
		topic, err := topicRepo.FindByID(ctx, *insight.TopicID)
		if err == nil {
			topicResponse = toTopicResponse(topic)
		}
	}

	return &types.InsightWithSources{
		Insight:        *insight,
		SourceContents: sourceContents,
		Topic:          topicResponse,
	}, nil
}

// ListInsights retrieves all insights for an organization
func ListInsights(ctx context.Context, orgID uuid.UUID, page, limit int) ([]types.InsightResponse, int64, error) {
	repo := getInsightRepo()

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	insights, total, err := repo.FindByOrganizationID(ctx, orgID, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	result := make([]types.InsightResponse, len(insights))
	for i, ins := range insights {
		result[i] = *toInsightResponse(&ins)
	}
	return result, total, nil
}

// ListAllInsights retrieves all insights with pagination (no org filter)
func ListAllInsights(ctx context.Context, page, limit int) ([]types.InsightResponse, int64, error) {
	repo := getInsightRepo()

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	insights, total, err := repo.List(ctx, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	result := make([]types.InsightResponse, len(insights))
	for i, ins := range insights {
		result[i] = *toInsightResponse(&ins)
	}
	return result, total, nil
}

// ListInsightsByTopic retrieves all insights for a topic
func ListInsightsByTopic(ctx context.Context, topicID uuid.UUID, page, limit int) ([]types.InsightResponse, int64, error) {
	repo := getInsightRepo()

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	insights, total, err := repo.FindByTopicID(ctx, topicID, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	result := make([]types.InsightResponse, len(insights))
	for i, ins := range insights {
		result[i] = *toInsightResponse(&ins)
	}
	return result, total, nil
}

// ListUnreadInsights retrieves unread insights for an organization
func ListUnreadInsights(ctx context.Context, orgID uuid.UUID, limit int) ([]types.InsightResponse, error) {
	repo := getInsightRepo()

	insights, err := repo.FindUnread(ctx, orgID, limit)
	if err != nil {
		return nil, err
	}

	result := make([]types.InsightResponse, len(insights))
	for i, ins := range insights {
		result[i] = *toInsightResponse(&ins)
	}
	return result, nil
}

// SearchInsights performs semantic search for insights
func SearchInsights(ctx context.Context, req types.InsightSearchRequest) ([]types.InsightResponse, error) {
	repo := getInsightRepo()
	embeddingService := getEmbeddingService()

	// Generate query embedding
	embedding, err := embeddingService.GenerateEmbedding(ctx, req.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	limit := req.Limit
	if limit < 1 || limit > 50 {
		limit = 10
	}

	insights, err := repo.SearchSimilar(ctx, embedding.Slice(), limit)
	if err != nil {
		return nil, err
	}

	result := make([]types.InsightResponse, len(insights))
	for i, ins := range insights {
		result[i] = *toInsightResponse(&ins)
	}
	return result, nil
}

// SearchInsightsByOrg performs semantic search for insights within an organization
func SearchInsightsByOrg(ctx context.Context, orgID uuid.UUID, query string, limit int) ([]types.InsightResponse, error) {
	repo := getInsightRepo()
	embeddingService := getEmbeddingService()

	// Generate query embedding
	embedding, err := embeddingService.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	if limit < 1 || limit > 50 {
		limit = 10
	}

	insights, err := repo.SearchSimilarByOrg(ctx, orgID, embedding.Slice(), limit)
	if err != nil {
		return nil, err
	}

	result := make([]types.InsightResponse, len(insights))
	for i, ins := range insights {
		result[i] = *toInsightResponse(&ins)
	}
	return result, nil
}

// CreateInsight creates a new insight
func CreateInsight(ctx context.Context, orgID *uuid.UUID, topicID *uuid.UUID, title, summary, content string, keyPoints []string, sourceContentIDs []uuid.UUID, periodStart, periodEnd *time.Time) (*types.InsightResponse, error) {
	repo := getInsightRepo()
	embeddingService := getEmbeddingService()

	// Generate embedding from title + summary
	embeddingText := title + " " + summary
	embedding, err := embeddingService.GenerateEmbedding(ctx, embeddingText)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	insight := &types.Insight{
		ID:               uuid.New(),
		OrganizationID:   orgID,
		TopicID:          topicID,
		Title:            title,
		Summary:          summary,
		Content:          &content,
		KeyPoints:        keyPoints,
		SourceContentIDs: sourceContentIDs,
		Embedding:        embedding.Slice(),
		GeneratedAt:      time.Now(),
		PeriodStart:      periodStart,
		PeriodEnd:        periodEnd,
		IsRead:           false,
		IsPinned:         false,
		IsUsedInArticle:  false,
	}

	if err := repo.Save(ctx, insight); err != nil {
		return nil, err
	}

	// Update topic's last insight time
	if topicID != nil {
		topicRepo := getTopicRepo()
		_ = topicRepo.UpdateLastInsightAt(ctx, *topicID, time.Now())
	}

	return toInsightResponse(insight), nil
}

// MarkInsightAsRead marks an insight as read
func MarkInsightAsRead(ctx context.Context, id uuid.UUID) error {
	repo := getInsightRepo()
	return repo.MarkAsRead(ctx, id)
}

// ToggleInsightPinned toggles the pinned status of an insight
func ToggleInsightPinned(ctx context.Context, id uuid.UUID) error {
	repo := getInsightRepo()
	return repo.TogglePinned(ctx, id)
}

// MarkInsightAsUsedInArticle marks an insight as used in an article
func MarkInsightAsUsedInArticle(ctx context.Context, id uuid.UUID) error {
	repo := getInsightRepo()
	return repo.MarkAsUsedInArticle(ctx, id)
}

// DeleteInsight removes an insight by its ID
func DeleteInsight(ctx context.Context, id uuid.UUID) error {
	repo := getInsightRepo()
	return repo.Delete(ctx, id)
}

// CountUnreadInsights counts unread insights for an organization
func CountUnreadInsights(ctx context.Context, orgID uuid.UUID) (int64, error) {
	repo := getInsightRepo()
	return repo.CountUnread(ctx, orgID)
}

// CountAllUnreadInsights returns the total count of unread insights (no org filter)
func CountAllUnreadInsights(ctx context.Context) (int64, error) {
	repo := getInsightRepo()
	return repo.CountAllUnread(ctx)
}

// =============================================================================
// Topic Functions
// =============================================================================

// GetTopicByID retrieves a topic by its ID
func GetTopicByID(ctx context.Context, id uuid.UUID) (*types.InsightTopicResponse, error) {
	repo := getTopicRepo()
	topic, err := repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toTopicResponse(topic), nil
}

// ListTopics retrieves all topics for an organization
func ListTopics(ctx context.Context, orgID uuid.UUID) ([]types.InsightTopicResponse, error) {
	repo := getTopicRepo()
	topics, err := repo.FindByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	result := make([]types.InsightTopicResponse, len(topics))
	for i, t := range topics {
		result[i] = *toTopicResponse(&t)
	}
	return result, nil
}

// ListAllTopics retrieves all topics
func ListAllTopics(ctx context.Context) ([]types.InsightTopicResponse, error) {
	repo := getTopicRepo()
	topics, err := repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]types.InsightTopicResponse, len(topics))
	for i, t := range topics {
		result[i] = *toTopicResponse(&t)
	}
	return result, nil
}

// CreateTopic creates a new topic with embedding
func CreateTopic(ctx context.Context, orgID *uuid.UUID, req types.InsightTopicCreateRequest) (*types.InsightTopicResponse, error) {
	repo := getTopicRepo()
	embeddingService := getEmbeddingService()

	// Generate embedding from name + description + keywords
	embeddingText := buildTopicEmbeddingText(req.Name, req.Description, req.Keywords)
	embedding, err := embeddingService.GenerateEmbedding(ctx, embeddingText)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	topic := &types.InsightTopic{
		ID:              uuid.New(),
		OrganizationID:  orgID,
		Name:            req.Name,
		Description:     req.Description,
		Keywords:        req.Keywords,
		Embedding:       embedding.Slice(),
		IsAutoGenerated: false,
		ContentCount:    0,
		Color:           req.Color,
		Icon:            req.Icon,
	}

	if err := repo.Save(ctx, topic); err != nil {
		return nil, err
	}

	return toTopicResponse(topic), nil
}

// UpdateTopic updates an existing topic
func UpdateTopic(ctx context.Context, id uuid.UUID, req types.InsightTopicUpdateRequest) (*types.InsightTopicResponse, error) {
	repo := getTopicRepo()
	embeddingService := getEmbeddingService()

	topic, err := repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	needsEmbeddingUpdate := false

	if req.Name != nil {
		topic.Name = *req.Name
		needsEmbeddingUpdate = true
	}
	if req.Description != nil {
		topic.Description = req.Description
		needsEmbeddingUpdate = true
	}
	if req.Keywords != nil {
		topic.Keywords = req.Keywords
		needsEmbeddingUpdate = true
	}
	if req.Color != nil {
		topic.Color = req.Color
	}
	if req.Icon != nil {
		topic.Icon = req.Icon
	}

	if needsEmbeddingUpdate {
		embeddingText := buildTopicEmbeddingText(topic.Name, topic.Description, topic.Keywords)
		embedding, err := embeddingService.GenerateEmbedding(ctx, embeddingText)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding: %w", err)
		}
		topic.Embedding = embedding.Slice()
	}

	if err := repo.Update(ctx, topic); err != nil {
		return nil, err
	}

	return toTopicResponse(topic), nil
}

// DeleteTopic removes a topic by its ID
func DeleteTopic(ctx context.Context, id uuid.UUID) error {
	repo := getTopicRepo()
	return repo.Delete(ctx, id)
}

// MatchContentToTopics finds matching topics for content based on embedding similarity
func MatchContentToTopics(ctx context.Context, contentID uuid.UUID, embedding []float32, threshold float64) ([]types.ContentTopicMatch, error) {
	topicRepo := getTopicRepo()
	matchRepo := getContentTopicMatchRepo()

	// Find similar topics
	topics, scores, err := topicRepo.SearchSimilar(ctx, embedding, 10, threshold)
	if err != nil {
		return nil, err
	}

	if len(topics) == 0 {
		return nil, nil
	}

	// Create matches
	matches := make([]types.ContentTopicMatch, len(topics))
	for i, topic := range topics {
		matches[i] = types.ContentTopicMatch{
			ID:              uuid.New(),
			ContentID:       contentID,
			TopicID:         topic.ID,
			SimilarityScore: scores[i],
			IsPrimary:       i == 0, // First match is primary (highest score)
		}
	}

	// Save matches
	if err := matchRepo.SaveBatch(ctx, matches); err != nil {
		return nil, err
	}

	// Update topic content counts
	for _, topic := range topics {
		count, _ := matchRepo.CountByTopicID(ctx, topic.ID)
		_ = topicRepo.UpdateContentCount(ctx, topic.ID, int(count))
	}

	return matches, nil
}

// =============================================================================
// Crawled Content Functions
// =============================================================================

// SearchCrawledContent performs semantic search for crawled content
func SearchCrawledContent(ctx context.Context, query string, limit int) ([]types.CrawledContentResponse, error) {
	contentRepo := getCrawledContentRepo()
	embeddingService := getEmbeddingService()

	// Generate query embedding
	embedding, err := embeddingService.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	if limit < 1 || limit > 50 {
		limit = 10
	}

	contents, err := contentRepo.SearchSimilar(ctx, embedding.Slice(), limit)
	if err != nil {
		return nil, err
	}

	result := make([]types.CrawledContentResponse, len(contents))
	for i, c := range contents {
		result[i] = types.CrawledContentResponse{
			ID:           c.ID,
			DataSourceID: c.DataSourceID,
			URL:          c.URL,
			Title:        c.Title,
			Content:      c.Content,
			Summary:      c.Summary,
			Author:       c.Author,
			PublishedAt:  c.PublishedAt,
			MetaData:     c.MetaData,
			CreatedAt:    c.CreatedAt,
		}
	}
	return result, nil
}

// SearchCrawledContentByOrg performs semantic search for crawled content within an organization
func SearchCrawledContentByOrg(ctx context.Context, orgID uuid.UUID, query string, limit int) ([]types.CrawledContentResponse, error) {
	contentRepo := getCrawledContentRepo()
	embeddingService := getEmbeddingService()

	// Generate query embedding
	embedding, err := embeddingService.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	if limit < 1 || limit > 50 {
		limit = 10
	}

	contents, err := contentRepo.SearchSimilarByOrg(ctx, orgID, embedding.Slice(), limit)
	if err != nil {
		return nil, err
	}

	result := make([]types.CrawledContentResponse, len(contents))
	for i, c := range contents {
		result[i] = types.CrawledContentResponse{
			ID:           c.ID,
			DataSourceID: c.DataSourceID,
			URL:          c.URL,
			Title:        c.Title,
			Content:      c.Content,
			Summary:      c.Summary,
			Author:       c.Author,
			PublishedAt:  c.PublishedAt,
			MetaData:     c.MetaData,
			CreatedAt:    c.CreatedAt,
		}
	}
	return result, nil
}

// GetRecentCrawledContent retrieves recent crawled content for an organization
func GetRecentCrawledContent(ctx context.Context, orgID uuid.UUID, limit int) ([]types.CrawledContentResponse, error) {
	contentRepo := getCrawledContentRepo()

	contents, err := contentRepo.FindRecentByOrg(ctx, orgID, limit)
	if err != nil {
		return nil, err
	}

	result := make([]types.CrawledContentResponse, len(contents))
	for i, c := range contents {
		result[i] = types.CrawledContentResponse{
			ID:           c.ID,
			DataSourceID: c.DataSourceID,
			URL:          c.URL,
			Title:        c.Title,
			Content:      c.Content,
			Summary:      c.Summary,
			Author:       c.Author,
			PublishedAt:  c.PublishedAt,
			MetaData:     c.MetaData,
			CreatedAt:    c.CreatedAt,
		}
	}
	return result, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

func buildTopicEmbeddingText(name string, description *string, keywords []string) string {
	parts := []string{name}
	if description != nil && *description != "" {
		parts = append(parts, *description)
	}
	if len(keywords) > 0 {
		parts = append(parts, strings.Join(keywords, " "))
	}
	return strings.Join(parts, " ")
}

func toInsightResponse(ins *types.Insight) *types.InsightResponse {
	return &types.InsightResponse{
		ID:               ins.ID,
		OrganizationID:   ins.OrganizationID,
		TopicID:          ins.TopicID,
		Title:            ins.Title,
		Summary:          ins.Summary,
		Content:          ins.Content,
		KeyPoints:        ins.KeyPoints,
		SourceContentIDs: ins.SourceContentIDs,
		GeneratedAt:      ins.GeneratedAt,
		PeriodStart:      ins.PeriodStart,
		PeriodEnd:        ins.PeriodEnd,
		IsRead:           ins.IsRead,
		IsPinned:         ins.IsPinned,
		IsUsedInArticle:  ins.IsUsedInArticle,
		MetaData:         ins.MetaData,
	}
}

func toTopicResponse(topic *types.InsightTopic) *types.InsightTopicResponse {
	return &types.InsightTopicResponse{
		ID:              topic.ID,
		OrganizationID:  topic.OrganizationID,
		Name:            topic.Name,
		Description:     topic.Description,
		Keywords:        topic.Keywords,
		IsAutoGenerated: topic.IsAutoGenerated,
		ContentCount:    topic.ContentCount,
		LastInsightAt:   topic.LastInsightAt,
		Color:           topic.Color,
		Icon:            topic.Icon,
		CreatedAt:       topic.CreatedAt,
		UpdatedAt:       topic.UpdatedAt,
	}
}
