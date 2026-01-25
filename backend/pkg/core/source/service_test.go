package source_test

import (
	"context"
	"testing"

	"backend/pkg/core"
	"backend/pkg/core/source"
	"backend/testutil/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestService_GetByID(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockSourceStore)
	svc := source.NewService(mockStore)

	t.Run("returns source when found", func(t *testing.T) {
		sourceID := uuid.New()
		expected := &source.Source{ID: sourceID, Title: "Test Source"}
		mockStore.On("FindByID", ctx, sourceID).Return(expected, nil).Once()

		result, err := svc.GetByID(ctx, sourceID)

		assert.NoError(t, err)
		assert.Equal(t, expected, result)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		sourceID := uuid.New()
		mockStore.On("FindByID", ctx, sourceID).Return(nil, core.ErrNotFound).Once()

		result, err := svc.GetByID(ctx, sourceID)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, core.ErrNotFound)
		mockStore.AssertExpectations(t)
	})
}

func TestService_GetByArticleID(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockSourceStore)
	svc := source.NewService(mockStore)

	t.Run("returns sources for article", func(t *testing.T) {
		articleID := uuid.New()
		sources := []source.Source{
			{ID: uuid.New(), ArticleID: articleID, Title: "Source 1"},
			{ID: uuid.New(), ArticleID: articleID, Title: "Source 2"},
		}
		mockStore.On("FindByArticleID", ctx, articleID).Return(sources, nil).Once()

		result, err := svc.GetByArticleID(ctx, articleID)

		assert.NoError(t, err)
		assert.Equal(t, sources, result)
		mockStore.AssertExpectations(t)
	})
}

func TestService_Create(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockSourceStore)
	svc := source.NewService(mockStore)

	t.Run("creates source successfully", func(t *testing.T) {
		articleID := uuid.New()
		req := source.CreateRequest{
			ArticleID:  articleID,
			Title:      "New Source",
			Content:    "Source content",
			URL:        "https://example.com",
			SourceType: "web",
		}
		mockStore.On("Save", ctx, mock.AnythingOfType("*source.Source")).Return(nil).Once()

		result, err := svc.Create(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, req.Title, result.Title)
		assert.Equal(t, req.URL, result.URL)
		assert.Equal(t, "web", result.SourceType)
		mockStore.AssertExpectations(t)
	})

	t.Run("defaults source type to web", func(t *testing.T) {
		articleID := uuid.New()
		req := source.CreateRequest{
			ArticleID: articleID,
			Title:     "Source",
		}
		mockStore.On("Save", ctx, mock.AnythingOfType("*source.Source")).Return(nil).Once()

		result, err := svc.Create(ctx, req)

		assert.NoError(t, err)
		assert.Equal(t, "web", result.SourceType)
		mockStore.AssertExpectations(t)
	})
}

func TestService_Update(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockSourceStore)
	svc := source.NewService(mockStore)

	t.Run("updates source successfully", func(t *testing.T) {
		sourceID := uuid.New()
		existing := &source.Source{
			ID:    sourceID,
			Title: "Old Title",
			URL:   "https://old.com",
		}
		newTitle := "New Title"
		req := source.UpdateRequest{
			Title: &newTitle,
		}
		mockStore.On("FindByID", ctx, sourceID).Return(existing, nil).Once()
		mockStore.On("Update", ctx, existing).Return(nil).Once()

		result, err := svc.Update(ctx, sourceID, req)

		assert.NoError(t, err)
		assert.Equal(t, newTitle, result.Title)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error when source not found", func(t *testing.T) {
		sourceID := uuid.New()
		req := source.UpdateRequest{}
		mockStore.On("FindByID", ctx, sourceID).Return(nil, core.ErrNotFound).Once()

		result, err := svc.Update(ctx, sourceID, req)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, core.ErrNotFound)
		mockStore.AssertExpectations(t)
	})
}

func TestService_Delete(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockSourceStore)
	svc := source.NewService(mockStore)

	t.Run("deletes source successfully", func(t *testing.T) {
		sourceID := uuid.New()
		mockStore.On("Delete", ctx, sourceID).Return(nil).Once()

		err := svc.Delete(ctx, sourceID)

		assert.NoError(t, err)
		mockStore.AssertExpectations(t)
	})
}

func TestService_SearchSimilar(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockSourceStore)
	svc := source.NewService(mockStore)

	t.Run("returns similar sources", func(t *testing.T) {
		articleID := uuid.New()
		embedding := []float32{0.1, 0.2, 0.3}
		sources := []source.Source{
			{ID: uuid.New(), Title: "Similar 1"},
		}
		mockStore.On("SearchSimilar", ctx, articleID, embedding, 5).Return(sources, nil).Once()

		result, err := svc.SearchSimilar(ctx, articleID, embedding, 5)

		assert.NoError(t, err)
		assert.Equal(t, sources, result)
		mockStore.AssertExpectations(t)
	})

	t.Run("uses default limit when invalid", func(t *testing.T) {
		articleID := uuid.New()
		embedding := []float32{0.1, 0.2, 0.3}
		sources := []source.Source{}
		mockStore.On("SearchSimilar", ctx, articleID, embedding, 5).Return(sources, nil).Once()

		result, err := svc.SearchSimilar(ctx, articleID, embedding, 0)

		assert.NoError(t, err)
		mockStore.AssertExpectations(t)
		assert.NotNil(t, result)
	})
}

func TestService_UpdateEmbedding(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockSourceStore)
	svc := source.NewService(mockStore)

	t.Run("updates embedding successfully", func(t *testing.T) {
		sourceID := uuid.New()
		existing := &source.Source{ID: sourceID}
		embedding := []float32{0.1, 0.2, 0.3}

		mockStore.On("FindByID", ctx, sourceID).Return(existing, nil).Once()
		mockStore.On("Update", ctx, existing).Return(nil).Once()

		err := svc.UpdateEmbedding(ctx, sourceID, embedding)

		assert.NoError(t, err)
		assert.Equal(t, embedding, existing.Embedding)
		mockStore.AssertExpectations(t)
	})
}
