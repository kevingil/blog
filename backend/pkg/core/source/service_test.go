package source_test

import (
	"context"
	"testing"
	"time"

	"backend/pkg/core"
	"backend/pkg/core/source"
	"backend/pkg/types"
	"backend/testutil/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestService_GetByID(t *testing.T) {
	ctx := context.Background()

	t.Run("returns source when found", func(t *testing.T) {
		mockSourceStore := new(mocks.MockSourceRepository)
		svc := source.NewService(mockSourceStore, nil)

		sourceID := uuid.New()
		articleID := uuid.New()
		testSource := &types.Source{
			ID:         sourceID,
			ArticleID:  articleID,
			Title:      "Test Source",
			Content:    "Test content for the source",
			URL:        "https://example.com/article",
			SourceType: "web",
			CreatedAt:  time.Now(),
		}

		mockSourceStore.On("FindByID", ctx, sourceID).Return(testSource, nil).Once()

		result, err := svc.GetByID(ctx, sourceID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, sourceID, result.ID)
		assert.Equal(t, "Test Source", result.Title)
		assert.Equal(t, "https://example.com/article", result.URL)
		mockSourceStore.AssertExpectations(t)
	})

	t.Run("returns error when source not found", func(t *testing.T) {
		mockSourceStore := new(mocks.MockSourceRepository)
		svc := source.NewService(mockSourceStore, nil)

		sourceID := uuid.New()
		mockSourceStore.On("FindByID", ctx, sourceID).Return(nil, core.ErrNotFound).Once()

		result, err := svc.GetByID(ctx, sourceID)

		assert.ErrorIs(t, err, core.ErrNotFound)
		assert.Nil(t, result)
		mockSourceStore.AssertExpectations(t)
	})
}

func TestService_GetByArticleID(t *testing.T) {
	ctx := context.Background()

	t.Run("returns sources for article", func(t *testing.T) {
		mockSourceStore := new(mocks.MockSourceRepository)
		svc := source.NewService(mockSourceStore, nil)

		articleID := uuid.New()
		testSources := []types.Source{
			{
				ID:         uuid.New(),
				ArticleID:  articleID,
				Title:      "Source 1",
				Content:    "Content 1",
				URL:        "https://example.com/1",
				SourceType: "web",
				CreatedAt:  time.Now(),
			},
			{
				ID:         uuid.New(),
				ArticleID:  articleID,
				Title:      "Source 2",
				Content:    "Content 2",
				URL:        "https://example.com/2",
				SourceType: "web",
				CreatedAt:  time.Now(),
			},
		}

		mockSourceStore.On("FindByArticleID", ctx, articleID).Return(testSources, nil).Once()

		result, err := svc.GetByArticleID(ctx, articleID)

		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "Source 1", result[0].Title)
		assert.Equal(t, "Source 2", result[1].Title)
		mockSourceStore.AssertExpectations(t)
	})

	t.Run("returns empty slice when no sources found", func(t *testing.T) {
		mockSourceStore := new(mocks.MockSourceRepository)
		svc := source.NewService(mockSourceStore, nil)

		articleID := uuid.New()
		mockSourceStore.On("FindByArticleID", ctx, articleID).Return([]types.Source{}, nil).Once()

		result, err := svc.GetByArticleID(ctx, articleID)

		assert.NoError(t, err)
		assert.Empty(t, result)
		mockSourceStore.AssertExpectations(t)
	})
}

func TestService_List(t *testing.T) {
	ctx := context.Background()

	t.Run("returns paginated sources with article metadata", func(t *testing.T) {
		mockSourceStore := new(mocks.MockSourceRepository)
		svc := source.NewService(mockSourceStore, nil)

		articleID := uuid.New()
		sourcesWithArticle := []source.SourceWithArticle{
			{
				Source: types.Source{
					ID:         uuid.New(),
					ArticleID:  articleID,
					Title:      "Source 1",
					Content:    "Content 1",
					SourceType: "web",
					CreatedAt:  time.Now(),
				},
				ArticleTitle: "Test Article",
				ArticleSlug:  "test-article",
			},
			{
				Source: types.Source{
					ID:         uuid.New(),
					ArticleID:  articleID,
					Title:      "Source 2",
					Content:    "Content 2",
					SourceType: "pdf",
					CreatedAt:  time.Now(),
				},
				ArticleTitle: "Test Article",
				ArticleSlug:  "test-article",
			},
		}

		expectedOpts := source.SourceListOptions{
			Page:    1,
			PerPage: 20,
		}
		mockSourceStore.On("List", ctx, expectedOpts).Return(sourcesWithArticle, int64(2), nil).Once()

		result, err := svc.List(ctx, 1, 20)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Sources, 2)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 1, result.TotalPages)
		assert.Equal(t, "Source 1", result.Sources[0].Source.Title)
		assert.Equal(t, "Test Article", result.Sources[0].ArticleTitle)
		mockSourceStore.AssertExpectations(t)
	})

	t.Run("normalizes invalid page and limit values", func(t *testing.T) {
		mockSourceStore := new(mocks.MockSourceRepository)
		svc := source.NewService(mockSourceStore, nil)

		// Pass invalid values (page=0, limit=0)
		expectedOpts := source.SourceListOptions{
			Page:    1,  // normalized from 0
			PerPage: 20, // normalized from 0
		}
		mockSourceStore.On("List", ctx, expectedOpts).Return([]source.SourceWithArticle{}, int64(0), nil).Once()

		result, err := svc.List(ctx, 0, 0)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, result.Page)
		mockSourceStore.AssertExpectations(t)
	})

	t.Run("caps limit at 100", func(t *testing.T) {
		mockSourceStore := new(mocks.MockSourceRepository)
		svc := source.NewService(mockSourceStore, nil)

		// Pass limit > 100
		expectedOpts := source.SourceListOptions{
			Page:    1,
			PerPage: 20, // normalized from 150 (over limit)
		}
		mockSourceStore.On("List", ctx, expectedOpts).Return([]source.SourceWithArticle{}, int64(0), nil).Once()

		result, err := svc.List(ctx, 1, 150)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		mockSourceStore.AssertExpectations(t)
	})

	t.Run("calculates total pages correctly", func(t *testing.T) {
		mockSourceStore := new(mocks.MockSourceRepository)
		svc := source.NewService(mockSourceStore, nil)

		expectedOpts := source.SourceListOptions{
			Page:    1,
			PerPage: 10,
		}
		// 25 total items with 10 per page = 3 pages
		mockSourceStore.On("List", ctx, expectedOpts).Return([]source.SourceWithArticle{}, int64(25), nil).Once()

		result, err := svc.List(ctx, 1, 10)

		assert.NoError(t, err)
		assert.Equal(t, 3, result.TotalPages)
		mockSourceStore.AssertExpectations(t)
	})
}

func TestService_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("deletes source successfully", func(t *testing.T) {
		mockSourceStore := new(mocks.MockSourceRepository)
		svc := source.NewService(mockSourceStore, nil)

		sourceID := uuid.New()
		mockSourceStore.On("Delete", ctx, sourceID).Return(nil).Once()

		err := svc.Delete(ctx, sourceID)

		assert.NoError(t, err)
		mockSourceStore.AssertExpectations(t)
	})

	t.Run("returns error when source not found", func(t *testing.T) {
		mockSourceStore := new(mocks.MockSourceRepository)
		svc := source.NewService(mockSourceStore, nil)

		sourceID := uuid.New()
		mockSourceStore.On("Delete", ctx, sourceID).Return(core.ErrNotFound).Once()

		err := svc.Delete(ctx, sourceID)

		assert.ErrorIs(t, err, core.ErrNotFound)
		mockSourceStore.AssertExpectations(t)
	})
}
