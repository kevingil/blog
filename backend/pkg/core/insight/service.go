package insight

import (
	"context"
	"fmt"
	"strings"
	"time"

	"backend/pkg/api/dto"
	"backend/pkg/types"

	"github.com/google/uuid"
)

// Service provides business logic for insights
type Service struct {
	insightStore      InsightStore
	topicStore        InsightTopicStore
	userStatusStore   UserInsightStatusStore
	contentStore      InsightCrawledContentStore
	matchStore        ContentTopicMatchStore
	embeddingService  EmbeddingGenerator
}

// NewService creates a new insight service with the provided stores
func NewService(
	insightStore InsightStore,
	topicStore InsightTopicStore,
	userStatusStore UserInsightStatusStore,
	contentStore InsightCrawledContentStore,
	matchStore ContentTopicMatchStore,
	embeddingService EmbeddingGenerator,
) *Service {
	return &Service{
		insightStore:     insightStore,
		topicStore:       topicStore,
		userStatusStore:  userStatusStore,
		contentStore:     contentStore,
		matchStore:       matchStore,
		embeddingService: embeddingService,
	}
}

// =============================================================================
// Insight Methods
// =============================================================================

// GetInsightByID retrieves an insight by its ID
func (s *Service) GetInsightByID(ctx context.Context, id uuid.UUID) (*dto.InsightResponse, error) {
	insight, err := s.insightStore.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toInsightResponse(insight), nil
}

// GetInsightWithSources retrieves an insight with its source content
func (s *Service) GetInsightWithSources(ctx context.Context, id uuid.UUID) (*dto.InsightWithSources, error) {
	insight, err := s.insightStore.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Get source contents
	var sourceContents []dto.CrawledContentResponse
	if len(insight.SourceContentIDs) > 0 {
		contents, err := s.contentStore.FindByIDs(ctx, insight.SourceContentIDs)
		if err != nil {
			return nil, err
		}
		sourceContents = make([]dto.CrawledContentResponse, len(contents))
		for i, c := range contents {
			sourceContents[i] = dto.CrawledContentResponse{
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
	var topicResponse *dto.InsightTopicResponse
	if insight.TopicID != nil {
		topic, err := s.topicStore.FindByID(ctx, *insight.TopicID)
		if err == nil {
			topicResponse = toTopicResponse(topic)
		}
	}

	return &dto.InsightWithSources{
		ID:               insight.ID,
		OrganizationID:   insight.OrganizationID,
		TopicID:          insight.TopicID,
		Title:            insight.Title,
		Summary:          insight.Summary,
		Content:          insight.Content,
		KeyPoints:        insight.KeyPoints,
		SourceContentIDs: insight.SourceContentIDs,
		GeneratedAt:      insight.GeneratedAt,
		PeriodStart:      insight.PeriodStart,
		PeriodEnd:        insight.PeriodEnd,
		IsRead:           insight.IsRead,
		IsPinned:         insight.IsPinned,
		IsUsedInArticle:  insight.IsUsedInArticle,
		MetaData:         insight.MetaData,
		SourceContents:   sourceContents,
		Topic:            topicResponse,
	}, nil
}

// ListInsights retrieves all insights for an organization
func (s *Service) ListInsights(ctx context.Context, orgID uuid.UUID, page, limit int) ([]dto.InsightResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	insights, total, err := s.insightStore.FindByOrganizationID(ctx, orgID, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	result := make([]dto.InsightResponse, len(insights))
	for i, ins := range insights {
		result[i] = *toInsightResponse(&ins)
	}
	return result, total, nil
}

// ListAllInsights retrieves all insights with pagination (no org filter)
func (s *Service) ListAllInsights(ctx context.Context, page, limit int) ([]dto.InsightResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	insights, total, err := s.insightStore.List(ctx, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	result := make([]dto.InsightResponse, len(insights))
	for i, ins := range insights {
		result[i] = *toInsightResponse(&ins)
	}
	return result, total, nil
}

// ListInsightsByTopic retrieves all insights for a topic
func (s *Service) ListInsightsByTopic(ctx context.Context, topicID uuid.UUID, page, limit int) ([]dto.InsightResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	insights, total, err := s.insightStore.FindByTopicID(ctx, topicID, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	result := make([]dto.InsightResponse, len(insights))
	for i, ins := range insights {
		result[i] = *toInsightResponse(&ins)
	}
	return result, total, nil
}

// ListUnreadInsights retrieves unread insights for an organization
func (s *Service) ListUnreadInsights(ctx context.Context, orgID uuid.UUID, limit int) ([]dto.InsightResponse, error) {
	insights, err := s.insightStore.FindUnread(ctx, orgID, limit)
	if err != nil {
		return nil, err
	}

	result := make([]dto.InsightResponse, len(insights))
	for i, ins := range insights {
		result[i] = *toInsightResponse(&ins)
	}
	return result, nil
}

// SearchInsights performs semantic search for insights
func (s *Service) SearchInsights(ctx context.Context, req dto.InsightSearchRequest) ([]dto.InsightResponse, error) {
	// Generate query embedding
	embedding, err := s.embeddingService.GenerateEmbedding(ctx, req.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	limit := req.Limit
	if limit < 1 || limit > 50 {
		limit = 10
	}

	insights, err := s.insightStore.SearchSimilar(ctx, embedding.Slice(), limit)
	if err != nil {
		return nil, err
	}

	result := make([]dto.InsightResponse, len(insights))
	for i, ins := range insights {
		result[i] = *toInsightResponse(&ins)
	}
	return result, nil
}

// SearchInsightsByOrg performs semantic search for insights within an organization
func (s *Service) SearchInsightsByOrg(ctx context.Context, orgID uuid.UUID, query string, limit int) ([]dto.InsightResponse, error) {
	// Generate query embedding
	embedding, err := s.embeddingService.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	if limit < 1 || limit > 50 {
		limit = 10
	}

	insights, err := s.insightStore.SearchSimilarByOrg(ctx, orgID, embedding.Slice(), limit)
	if err != nil {
		return nil, err
	}

	result := make([]dto.InsightResponse, len(insights))
	for i, ins := range insights {
		result[i] = *toInsightResponse(&ins)
	}
	return result, nil
}

// CreateInsight creates a new insight
func (s *Service) CreateInsight(ctx context.Context, orgID *uuid.UUID, topicID *uuid.UUID, title, summary, content string, keyPoints []string, sourceContentIDs []uuid.UUID, periodStart, periodEnd *time.Time) (*dto.InsightResponse, error) {
	// Generate embedding from title + summary
	embeddingText := title + " " + summary
	embedding, err := s.embeddingService.GenerateEmbedding(ctx, embeddingText)
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

	if err := s.insightStore.Save(ctx, insight); err != nil {
		return nil, err
	}

	// Update topic's last insight time
	if topicID != nil {
		_ = s.topicStore.UpdateLastInsightAt(ctx, *topicID, time.Now())
	}

	return toInsightResponse(insight), nil
}

// MarkInsightAsRead marks an insight as read (legacy - uses global flag)
func (s *Service) MarkInsightAsRead(ctx context.Context, id uuid.UUID) error {
	return s.insightStore.MarkAsRead(ctx, id)
}

// MarkInsightAsReadForUser marks an insight as read for a specific user
func (s *Service) MarkInsightAsReadForUser(ctx context.Context, userID, insightID uuid.UUID) error {
	return s.userStatusStore.MarkAsRead(ctx, userID, insightID)
}

// ToggleInsightPinnedForUser toggles the pinned status of an insight for a user
func (s *Service) ToggleInsightPinnedForUser(ctx context.Context, userID, insightID uuid.UUID) (bool, error) {
	return s.userStatusStore.TogglePinned(ctx, userID, insightID)
}

// MarkInsightAsUsedInArticleForUser marks an insight as used in an article for a user
func (s *Service) MarkInsightAsUsedInArticleForUser(ctx context.Context, userID, insightID uuid.UUID) error {
	return s.userStatusStore.MarkAsUsedInArticle(ctx, userID, insightID)
}

// GetUserInsightStatus retrieves the user's status for an insight
func (s *Service) GetUserInsightStatus(ctx context.Context, userID, insightID uuid.UUID) (*types.UserInsightStatus, error) {
	return s.userStatusStore.FindByUserAndInsight(ctx, userID, insightID)
}

// GetInsightWithUserStatus retrieves an insight with the user's status
func (s *Service) GetInsightWithUserStatus(ctx context.Context, userID, insightID uuid.UUID) (*dto.InsightWithUserStatus, error) {
	insight, err := s.insightStore.FindByID(ctx, insightID)
	if err != nil {
		return nil, err
	}

	status, err := s.userStatusStore.FindByUserAndInsight(ctx, userID, insightID)
	if err != nil {
		return nil, err
	}

	insightResp := toInsightResponse(insight)
	result := &dto.InsightWithUserStatus{
		InsightResponse: *insightResp,
	}

	if status != nil {
		result.UserStatus = &dto.UserInsightStatusResponse{
			InsightID:       status.InsightID,
			IsRead:          status.IsRead,
			IsPinned:        status.IsPinned,
			IsUsedInArticle: status.IsUsedInArticle,
			ReadAt:          status.ReadAt,
		}
	}

	return result, nil
}

// ListInsightsWithUserStatus retrieves insights with user-specific status
func (s *Service) ListInsightsWithUserStatus(ctx context.Context, userID uuid.UUID, page, limit int) ([]dto.InsightWithUserStatus, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	// Get all insights (global)
	insights, total, err := s.insightStore.List(ctx, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	// Get user status for all insights
	insightIDs := make([]uuid.UUID, len(insights))
	for i, ins := range insights {
		insightIDs[i] = ins.ID
	}

	statusMap, err := s.userStatusStore.GetStatusMapForInsights(ctx, userID, insightIDs)
	if err != nil {
		return nil, 0, err
	}

	// Combine insights with user status
	result := make([]dto.InsightWithUserStatus, len(insights))
	for i, ins := range insights {
		insightResp := toInsightResponse(&ins)
		result[i] = dto.InsightWithUserStatus{
			InsightResponse: *insightResp,
		}
		if status, ok := statusMap[ins.ID]; ok {
			result[i].UserStatus = &dto.UserInsightStatusResponse{
				InsightID:       status.InsightID,
				IsRead:          status.IsRead,
				IsPinned:        status.IsPinned,
				IsUsedInArticle: status.IsUsedInArticle,
				ReadAt:          status.ReadAt,
			}
		}
	}

	return result, total, nil
}

// CountUnreadInsightsForUser counts unread insights for a user
func (s *Service) CountUnreadInsightsForUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	// Get total insight count
	_, totalInsights, err := s.insightStore.List(ctx, 0, 1)
	if err != nil {
		return 0, err
	}

	// Get read count for user
	readCount, err := s.userStatusStore.CountUnreadByUserID(ctx, userID)
	if err != nil {
		return 0, err
	}

	// Unread = total - read statuses with is_read=true
	// Note: this is approximate - we count insights that don't have a read status
	// A more accurate count would require a LEFT JOIN query
	return totalInsights - readCount, nil
}

// ToggleInsightPinned toggles the pinned status of an insight
func (s *Service) ToggleInsightPinned(ctx context.Context, id uuid.UUID) error {
	return s.insightStore.TogglePinned(ctx, id)
}

// MarkInsightAsUsedInArticle marks an insight as used in an article
func (s *Service) MarkInsightAsUsedInArticle(ctx context.Context, id uuid.UUID) error {
	return s.insightStore.MarkAsUsedInArticle(ctx, id)
}

// DeleteInsight removes an insight by its ID
func (s *Service) DeleteInsight(ctx context.Context, id uuid.UUID) error {
	return s.insightStore.Delete(ctx, id)
}

// CountUnreadInsights counts unread insights for an organization
func (s *Service) CountUnreadInsights(ctx context.Context, orgID uuid.UUID) (int64, error) {
	return s.insightStore.CountUnread(ctx, orgID)
}

// CountAllUnreadInsights returns the total count of unread insights (no org filter)
func (s *Service) CountAllUnreadInsights(ctx context.Context) (int64, error) {
	return s.insightStore.CountAllUnread(ctx)
}

// =============================================================================
// Topic Methods
// =============================================================================

// GetTopicByID retrieves a topic by its ID
func (s *Service) GetTopicByID(ctx context.Context, id uuid.UUID) (*dto.InsightTopicResponse, error) {
	topic, err := s.topicStore.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toTopicResponse(topic), nil
}

// ListTopics retrieves all topics for an organization
func (s *Service) ListTopics(ctx context.Context, orgID uuid.UUID) ([]dto.InsightTopicResponse, error) {
	topics, err := s.topicStore.FindByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	result := make([]dto.InsightTopicResponse, len(topics))
	for i, t := range topics {
		result[i] = *toTopicResponse(&t)
	}
	return result, nil
}

// ListAllTopics retrieves all topics
func (s *Service) ListAllTopics(ctx context.Context) ([]dto.InsightTopicResponse, error) {
	topics, err := s.topicStore.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]dto.InsightTopicResponse, len(topics))
	for i, t := range topics {
		result[i] = *toTopicResponse(&t)
	}
	return result, nil
}

// CreateTopic creates a new topic with embedding
func (s *Service) CreateTopic(ctx context.Context, orgID *uuid.UUID, req dto.InsightTopicCreateRequest) (*dto.InsightTopicResponse, error) {
	// Generate embedding from name + description + keywords
	embeddingText := buildTopicEmbeddingText(req.Name, req.Description, req.Keywords)
	embedding, err := s.embeddingService.GenerateEmbedding(ctx, embeddingText)
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

	if err := s.topicStore.Save(ctx, topic); err != nil {
		return nil, err
	}

	return toTopicResponse(topic), nil
}

// UpdateTopic updates an existing topic
func (s *Service) UpdateTopic(ctx context.Context, id uuid.UUID, req dto.InsightTopicUpdateRequest) (*dto.InsightTopicResponse, error) {
	topic, err := s.topicStore.FindByID(ctx, id)
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
		embedding, err := s.embeddingService.GenerateEmbedding(ctx, embeddingText)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding: %w", err)
		}
		topic.Embedding = embedding.Slice()
	}

	if err := s.topicStore.Update(ctx, topic); err != nil {
		return nil, err
	}

	return toTopicResponse(topic), nil
}

// DeleteTopic removes a topic by its ID
func (s *Service) DeleteTopic(ctx context.Context, id uuid.UUID) error {
	return s.topicStore.Delete(ctx, id)
}

// MatchContentToTopics finds matching topics for content based on embedding similarity
func (s *Service) MatchContentToTopics(ctx context.Context, contentID uuid.UUID, embedding []float32, threshold float64) ([]types.ContentTopicMatch, error) {
	// Find similar topics
	topics, scores, err := s.topicStore.SearchSimilar(ctx, embedding, 10, threshold)
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
	if err := s.matchStore.SaveBatch(ctx, matches); err != nil {
		return nil, err
	}

	// Update topic content counts
	for _, topic := range topics {
		count, _ := s.matchStore.CountByTopicID(ctx, topic.ID)
		_ = s.topicStore.UpdateContentCount(ctx, topic.ID, int(count))
	}

	return matches, nil
}

// =============================================================================
// Crawled Content Methods
// =============================================================================

// SearchCrawledContent performs semantic search for crawled content
func (s *Service) SearchCrawledContent(ctx context.Context, query string, limit int) ([]dto.CrawledContentResponse, error) {
	// Generate query embedding
	embedding, err := s.embeddingService.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	if limit < 1 || limit > 50 {
		limit = 10
	}

	contents, err := s.contentStore.SearchSimilar(ctx, embedding.Slice(), limit)
	if err != nil {
		return nil, err
	}

	result := make([]dto.CrawledContentResponse, len(contents))
	for i, c := range contents {
		result[i] = dto.CrawledContentResponse{
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
func (s *Service) SearchCrawledContentByOrg(ctx context.Context, orgID uuid.UUID, query string, limit int) ([]dto.CrawledContentResponse, error) {
	// Generate query embedding
	embedding, err := s.embeddingService.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	if limit < 1 || limit > 50 {
		limit = 10
	}

	contents, err := s.contentStore.SearchSimilarByOrg(ctx, orgID, embedding.Slice(), limit)
	if err != nil {
		return nil, err
	}

	result := make([]dto.CrawledContentResponse, len(contents))
	for i, c := range contents {
		result[i] = dto.CrawledContentResponse{
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
func (s *Service) GetRecentCrawledContent(ctx context.Context, orgID uuid.UUID, limit int) ([]dto.CrawledContentResponse, error) {
	contents, err := s.contentStore.FindRecentByOrg(ctx, orgID, limit)
	if err != nil {
		return nil, err
	}

	result := make([]dto.CrawledContentResponse, len(contents))
	for i, c := range contents {
		result[i] = dto.CrawledContentResponse{
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

func toInsightResponse(ins *types.Insight) *dto.InsightResponse {
	return &dto.InsightResponse{
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

func toTopicResponse(topic *types.InsightTopic) *dto.InsightTopicResponse {
	return &dto.InsightTopicResponse{
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
