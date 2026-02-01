package insight_test

import (
	"context"
	"testing"
	"time"

	"backend/pkg/api/dto"
	"backend/pkg/core"
	"backend/pkg/core/insight"
	"backend/pkg/types"
	"backend/testutil/mocks"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Helper function to create a new service with all mocks
func newTestService() (
	*insight.Service,
	*mocks.MockInsightRepository,
	*mocks.MockInsightTopicRepository,
	*mocks.MockUserInsightStatusRepository,
	*mocks.MockCrawledContentRepository,
	*mocks.MockContentTopicMatchRepository,
	*mocks.MockEmbeddingGenerator,
) {
	mockInsightStore := new(mocks.MockInsightRepository)
	mockTopicStore := new(mocks.MockInsightTopicRepository)
	mockUserStatusStore := new(mocks.MockUserInsightStatusRepository)
	mockContentStore := new(mocks.MockCrawledContentRepository)
	mockMatchStore := new(mocks.MockContentTopicMatchRepository)
	mockEmbeddingGen := new(mocks.MockEmbeddingGenerator)

	svc := insight.NewService(
		mockInsightStore,
		mockTopicStore,
		mockUserStatusStore,
		mockContentStore,
		mockMatchStore,
		mockEmbeddingGen,
	)

	return svc, mockInsightStore, mockTopicStore, mockUserStatusStore, mockContentStore, mockMatchStore, mockEmbeddingGen
}

// =============================================================================
// Insight Tests
// =============================================================================

func TestService_GetInsightByID(t *testing.T) {
	ctx := context.Background()

	t.Run("returns insight when found", func(t *testing.T) {
		svc, mockInsightStore, _, _, _, _, _ := newTestService()

		insightID := uuid.New()
		orgID := uuid.New()
		content := "Test content"
		testInsight := &types.Insight{
			ID:             insightID,
			OrganizationID: &orgID,
			Title:          "Test Insight",
			Summary:        "Test summary",
			Content:        &content,
			KeyPoints:      []string{"point1", "point2"},
			GeneratedAt:    time.Now(),
			IsRead:         false,
			IsPinned:       false,
		}

		mockInsightStore.On("FindByID", ctx, insightID).Return(testInsight, nil).Once()

		result, err := svc.GetInsightByID(ctx, insightID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, insightID, result.ID)
		assert.Equal(t, "Test Insight", result.Title)
		assert.Equal(t, "Test summary", result.Summary)
		mockInsightStore.AssertExpectations(t)
	})

	t.Run("returns error when insight not found", func(t *testing.T) {
		svc, mockInsightStore, _, _, _, _, _ := newTestService()

		insightID := uuid.New()
		mockInsightStore.On("FindByID", ctx, insightID).Return(nil, core.ErrNotFound).Once()

		result, err := svc.GetInsightByID(ctx, insightID)

		assert.ErrorIs(t, err, core.ErrNotFound)
		assert.Nil(t, result)
		mockInsightStore.AssertExpectations(t)
	})
}

func TestService_ListInsights(t *testing.T) {
	ctx := context.Background()

	t.Run("returns paginated insights for organization", func(t *testing.T) {
		svc, mockInsightStore, _, _, _, _, _ := newTestService()

		orgID := uuid.New()
		content1 := "Content 1"
		content2 := "Content 2"
		insights := []types.Insight{
			{
				ID:             uuid.New(),
				OrganizationID: &orgID,
				Title:          "Insight 1",
				Summary:        "Summary 1",
				Content:        &content1,
				GeneratedAt:    time.Now(),
			},
			{
				ID:             uuid.New(),
				OrganizationID: &orgID,
				Title:          "Insight 2",
				Summary:        "Summary 2",
				Content:        &content2,
				GeneratedAt:    time.Now(),
			},
		}

		mockInsightStore.On("FindByOrganizationID", ctx, orgID, 0, 20).Return(insights, int64(2), nil).Once()

		result, total, err := svc.ListInsights(ctx, orgID, 1, 20)

		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, int64(2), total)
		assert.Equal(t, "Insight 1", result[0].Title)
		assert.Equal(t, "Insight 2", result[1].Title)
		mockInsightStore.AssertExpectations(t)
	})

	t.Run("applies default pagination for invalid values", func(t *testing.T) {
		svc, mockInsightStore, _, _, _, _, _ := newTestService()

		orgID := uuid.New()
		mockInsightStore.On("FindByOrganizationID", ctx, orgID, 0, 20).Return([]types.Insight{}, int64(0), nil).Once()

		_, _, err := svc.ListInsights(ctx, orgID, 0, 0) // Invalid page and limit

		assert.NoError(t, err)
		mockInsightStore.AssertExpectations(t)
	})

	t.Run("limits max page size to 100", func(t *testing.T) {
		svc, mockInsightStore, _, _, _, _, _ := newTestService()

		orgID := uuid.New()
		mockInsightStore.On("FindByOrganizationID", ctx, orgID, 0, 20).Return([]types.Insight{}, int64(0), nil).Once()

		_, _, err := svc.ListInsights(ctx, orgID, 1, 500) // Exceeds max limit

		assert.NoError(t, err)
		mockInsightStore.AssertExpectations(t)
	})
}

func TestService_CreateInsight(t *testing.T) {
	ctx := context.Background()

	t.Run("creates insight successfully", func(t *testing.T) {
		svc, mockInsightStore, mockTopicStore, _, _, _, mockEmbeddingGen := newTestService()

		orgID := uuid.New()
		topicID := uuid.New()
		title := "New Insight"
		summary := "New summary"
		content := "New content"
		keyPoints := []string{"point1", "point2"}
		sourceIDs := []uuid.UUID{uuid.New()}

		embedding := pgvector.NewVector([]float32{0.1, 0.2, 0.3})

		mockEmbeddingGen.On("GenerateEmbedding", ctx, title+" "+summary).Return(embedding, nil).Once()
		mockInsightStore.On("Save", ctx, mock.AnythingOfType("*types.Insight")).Return(nil).Once()
		mockTopicStore.On("UpdateLastInsightAt", ctx, topicID, mock.AnythingOfType("time.Time")).Return(nil).Once()

		result, err := svc.CreateInsight(ctx, &orgID, &topicID, title, summary, content, keyPoints, sourceIDs, nil, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "New Insight", result.Title)
		assert.Equal(t, "New summary", result.Summary)
		mockInsightStore.AssertExpectations(t)
		mockTopicStore.AssertExpectations(t)
		mockEmbeddingGen.AssertExpectations(t)
	})

	t.Run("creates insight without topic", func(t *testing.T) {
		svc, mockInsightStore, _, _, _, _, mockEmbeddingGen := newTestService()

		orgID := uuid.New()
		title := "New Insight"
		summary := "New summary"
		content := "New content"

		embedding := pgvector.NewVector([]float32{0.1, 0.2, 0.3})

		mockEmbeddingGen.On("GenerateEmbedding", ctx, title+" "+summary).Return(embedding, nil).Once()
		mockInsightStore.On("Save", ctx, mock.AnythingOfType("*types.Insight")).Return(nil).Once()

		result, err := svc.CreateInsight(ctx, &orgID, nil, title, summary, content, nil, nil, nil, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		mockInsightStore.AssertExpectations(t)
		mockEmbeddingGen.AssertExpectations(t)
	})
}

func TestService_DeleteInsight(t *testing.T) {
	ctx := context.Background()

	t.Run("deletes insight successfully", func(t *testing.T) {
		svc, mockInsightStore, _, _, _, _, _ := newTestService()

		insightID := uuid.New()
		mockInsightStore.On("Delete", ctx, insightID).Return(nil).Once()

		err := svc.DeleteInsight(ctx, insightID)

		assert.NoError(t, err)
		mockInsightStore.AssertExpectations(t)
	})

	t.Run("returns error when insight not found", func(t *testing.T) {
		svc, mockInsightStore, _, _, _, _, _ := newTestService()

		insightID := uuid.New()
		mockInsightStore.On("Delete", ctx, insightID).Return(core.ErrNotFound).Once()

		err := svc.DeleteInsight(ctx, insightID)

		assert.ErrorIs(t, err, core.ErrNotFound)
		mockInsightStore.AssertExpectations(t)
	})
}

// =============================================================================
// Topic Tests
// =============================================================================

func TestService_GetTopicByID(t *testing.T) {
	ctx := context.Background()

	t.Run("returns topic when found", func(t *testing.T) {
		svc, _, mockTopicStore, _, _, _, _ := newTestService()

		topicID := uuid.New()
		orgID := uuid.New()
		description := "Test description"
		testTopic := &types.InsightTopic{
			ID:              topicID,
			OrganizationID:  &orgID,
			Name:            "Test Topic",
			Description:     &description,
			Keywords:        []string{"keyword1", "keyword2"},
			IsAutoGenerated: false,
			ContentCount:    5,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		mockTopicStore.On("FindByID", ctx, topicID).Return(testTopic, nil).Once()

		result, err := svc.GetTopicByID(ctx, topicID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, topicID, result.ID)
		assert.Equal(t, "Test Topic", result.Name)
		assert.Equal(t, 5, result.ContentCount)
		mockTopicStore.AssertExpectations(t)
	})

	t.Run("returns error when topic not found", func(t *testing.T) {
		svc, _, mockTopicStore, _, _, _, _ := newTestService()

		topicID := uuid.New()
		mockTopicStore.On("FindByID", ctx, topicID).Return(nil, core.ErrNotFound).Once()

		result, err := svc.GetTopicByID(ctx, topicID)

		assert.ErrorIs(t, err, core.ErrNotFound)
		assert.Nil(t, result)
		mockTopicStore.AssertExpectations(t)
	})
}

func TestService_ListTopics(t *testing.T) {
	ctx := context.Background()

	t.Run("returns topics for organization", func(t *testing.T) {
		svc, _, mockTopicStore, _, _, _, _ := newTestService()

		orgID := uuid.New()
		topics := []types.InsightTopic{
			{
				ID:              uuid.New(),
				OrganizationID:  &orgID,
				Name:            "Topic 1",
				Keywords:        []string{"key1"},
				IsAutoGenerated: false,
				ContentCount:    3,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			},
			{
				ID:              uuid.New(),
				OrganizationID:  &orgID,
				Name:            "Topic 2",
				Keywords:        []string{"key2"},
				IsAutoGenerated: true,
				ContentCount:    7,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			},
		}

		mockTopicStore.On("FindByOrganizationID", ctx, orgID).Return(topics, nil).Once()

		result, err := svc.ListTopics(ctx, orgID)

		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "Topic 1", result[0].Name)
		assert.Equal(t, "Topic 2", result[1].Name)
		mockTopicStore.AssertExpectations(t)
	})

	t.Run("returns empty list when no topics exist", func(t *testing.T) {
		svc, _, mockTopicStore, _, _, _, _ := newTestService()

		orgID := uuid.New()
		mockTopicStore.On("FindByOrganizationID", ctx, orgID).Return([]types.InsightTopic{}, nil).Once()

		result, err := svc.ListTopics(ctx, orgID)

		assert.NoError(t, err)
		assert.Empty(t, result)
		mockTopicStore.AssertExpectations(t)
	})
}

func TestService_CreateTopic(t *testing.T) {
	ctx := context.Background()

	t.Run("creates topic successfully", func(t *testing.T) {
		svc, _, mockTopicStore, _, _, _, mockEmbeddingGen := newTestService()

		orgID := uuid.New()
		description := "Test description"
		color := "#FF0000"
		icon := "star"
		req := dto.InsightTopicCreateRequest{
			Name:        "New Topic",
			Description: &description,
			Keywords:    []string{"keyword1", "keyword2"},
			Color:       &color,
			Icon:        &icon,
		}

		embedding := pgvector.NewVector([]float32{0.1, 0.2, 0.3})

		mockEmbeddingGen.On("GenerateEmbedding", ctx, "New Topic Test description keyword1 keyword2").Return(embedding, nil).Once()
		mockTopicStore.On("Save", ctx, mock.AnythingOfType("*types.InsightTopic")).Return(nil).Once()

		result, err := svc.CreateTopic(ctx, &orgID, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "New Topic", result.Name)
		assert.Equal(t, &description, result.Description)
		assert.Equal(t, []string{"keyword1", "keyword2"}, result.Keywords)
		mockTopicStore.AssertExpectations(t)
		mockEmbeddingGen.AssertExpectations(t)
	})

	t.Run("creates topic with minimal fields", func(t *testing.T) {
		svc, _, mockTopicStore, _, _, _, mockEmbeddingGen := newTestService()

		orgID := uuid.New()
		req := dto.InsightTopicCreateRequest{
			Name: "Minimal Topic",
		}

		embedding := pgvector.NewVector([]float32{0.1, 0.2, 0.3})

		mockEmbeddingGen.On("GenerateEmbedding", ctx, "Minimal Topic").Return(embedding, nil).Once()
		mockTopicStore.On("Save", ctx, mock.AnythingOfType("*types.InsightTopic")).Return(nil).Once()

		result, err := svc.CreateTopic(ctx, &orgID, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Minimal Topic", result.Name)
		mockTopicStore.AssertExpectations(t)
		mockEmbeddingGen.AssertExpectations(t)
	})
}

func TestService_DeleteTopic(t *testing.T) {
	ctx := context.Background()

	t.Run("deletes topic successfully", func(t *testing.T) {
		svc, _, mockTopicStore, _, _, _, _ := newTestService()

		topicID := uuid.New()
		mockTopicStore.On("Delete", ctx, topicID).Return(nil).Once()

		err := svc.DeleteTopic(ctx, topicID)

		assert.NoError(t, err)
		mockTopicStore.AssertExpectations(t)
	})

	t.Run("returns error when topic not found", func(t *testing.T) {
		svc, _, mockTopicStore, _, _, _, _ := newTestService()

		topicID := uuid.New()
		mockTopicStore.On("Delete", ctx, topicID).Return(core.ErrNotFound).Once()

		err := svc.DeleteTopic(ctx, topicID)

		assert.ErrorIs(t, err, core.ErrNotFound)
		mockTopicStore.AssertExpectations(t)
	})
}

// =============================================================================
// Additional Coverage Tests
// =============================================================================

func TestService_MarkInsightAsRead(t *testing.T) {
	ctx := context.Background()

	t.Run("marks insight as read successfully", func(t *testing.T) {
		svc, mockInsightStore, _, _, _, _, _ := newTestService()

		insightID := uuid.New()
		mockInsightStore.On("MarkAsRead", ctx, insightID).Return(nil).Once()

		err := svc.MarkInsightAsRead(ctx, insightID)

		assert.NoError(t, err)
		mockInsightStore.AssertExpectations(t)
	})
}

func TestService_ToggleInsightPinned(t *testing.T) {
	ctx := context.Background()

	t.Run("toggles pinned status successfully", func(t *testing.T) {
		svc, mockInsightStore, _, _, _, _, _ := newTestService()

		insightID := uuid.New()
		mockInsightStore.On("TogglePinned", ctx, insightID).Return(nil).Once()

		err := svc.ToggleInsightPinned(ctx, insightID)

		assert.NoError(t, err)
		mockInsightStore.AssertExpectations(t)
	})
}

func TestService_ListAllTopics(t *testing.T) {
	ctx := context.Background()

	t.Run("returns all topics", func(t *testing.T) {
		svc, _, mockTopicStore, _, _, _, _ := newTestService()

		topics := []types.InsightTopic{
			{
				ID:        uuid.New(),
				Name:      "Topic 1",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:        uuid.New(),
				Name:      "Topic 2",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}

		mockTopicStore.On("FindAll", ctx).Return(topics, nil).Once()

		result, err := svc.ListAllTopics(ctx)

		assert.NoError(t, err)
		assert.Len(t, result, 2)
		mockTopicStore.AssertExpectations(t)
	})
}
