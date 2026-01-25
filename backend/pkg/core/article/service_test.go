package article_test

import (
	"context"
	"testing"

	"backend/pkg/core"
	"backend/pkg/core/article"
	"backend/testutil/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestService_GetByID(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockArticleStore)
	mockTagStore := new(mocks.MockTagStore)
	svc := article.NewService(mockStore, mockTagStore)

	t.Run("returns article when found", func(t *testing.T) {
		articleID := uuid.New()
		expected := &article.Article{ID: articleID, Title: "Test Article"}
		mockStore.On("FindByID", ctx, articleID).Return(expected, nil).Once()

		result, err := svc.GetByID(ctx, articleID)

		assert.NoError(t, err)
		assert.Equal(t, expected, result)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		articleID := uuid.New()
		mockStore.On("FindByID", ctx, articleID).Return(nil, core.ErrNotFound).Once()

		result, err := svc.GetByID(ctx, articleID)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, core.ErrNotFound)
		mockStore.AssertExpectations(t)
	})
}

func TestService_GetBySlug(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockArticleStore)
	mockTagStore := new(mocks.MockTagStore)
	svc := article.NewService(mockStore, mockTagStore)

	t.Run("returns article when found", func(t *testing.T) {
		slug := "test-article"
		expected := &article.Article{ID: uuid.New(), Slug: slug}
		mockStore.On("FindBySlug", ctx, slug).Return(expected, nil).Once()

		result, err := svc.GetBySlug(ctx, slug)

		assert.NoError(t, err)
		assert.Equal(t, expected, result)
		mockStore.AssertExpectations(t)
	})
}

func TestService_List(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockArticleStore)
	mockTagStore := new(mocks.MockTagStore)
	svc := article.NewService(mockStore, mockTagStore)

	t.Run("returns paginated list", func(t *testing.T) {
		articles := []article.Article{
			{ID: uuid.New(), Title: "Article 1"},
			{ID: uuid.New(), Title: "Article 2"},
		}
		opts := article.ListOptions{Page: 1, PerPage: 10, IsDraft: nil, AuthorID: nil}
		mockStore.On("List", ctx, opts).Return(articles, int64(2), nil).Once()

		result, err := svc.List(ctx, 1, 10, nil, nil)

		assert.NoError(t, err)
		assert.Equal(t, articles, result.Articles)
		assert.Equal(t, int64(2), result.Total)
		mockStore.AssertExpectations(t)
	})

	t.Run("uses default values for invalid pagination", func(t *testing.T) {
		articles := []article.Article{}
		opts := article.ListOptions{Page: 1, PerPage: 20, IsDraft: nil, AuthorID: nil}
		mockStore.On("List", ctx, opts).Return(articles, int64(0), nil).Once()

		result, err := svc.List(ctx, 0, 0, nil, nil)

		assert.NoError(t, err)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 20, result.PerPage)
		mockStore.AssertExpectations(t)
	})
}

func TestService_Create(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockArticleStore)
	mockTagStore := new(mocks.MockTagStore)
	svc := article.NewService(mockStore, mockTagStore)

	t.Run("creates article successfully", func(t *testing.T) {
		req := article.CreateRequest{
			Title:    "New Article",
			Content:  "Content here",
			Slug:     "new-article",
			AuthorID: uuid.New(),
			IsDraft:  false,
		}
		mockStore.On("FindBySlug", ctx, req.Slug).Return(nil, core.ErrNotFound).Once()
		mockStore.On("Save", ctx, mock.AnythingOfType("*article.Article")).Return(nil).Once()

		result, err := svc.Create(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, req.Title, result.Title)
		assert.Equal(t, req.Slug, result.Slug)
		assert.NotNil(t, result.PublishedAt) // Not a draft, should have published date
		mockStore.AssertExpectations(t)
	})

	t.Run("creates draft article", func(t *testing.T) {
		req := article.CreateRequest{
			Title:    "Draft Article",
			Content:  "Draft content",
			Slug:     "draft-article",
			AuthorID: uuid.New(),
			IsDraft:  true,
		}
		mockStore.On("FindBySlug", ctx, req.Slug).Return(nil, core.ErrNotFound).Once()
		mockStore.On("Save", ctx, mock.AnythingOfType("*article.Article")).Return(nil).Once()

		result, err := svc.Create(ctx, req)

		assert.NoError(t, err)
		assert.True(t, result.IsDraft)
		assert.Nil(t, result.PublishedAt)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error when slug already exists", func(t *testing.T) {
		req := article.CreateRequest{
			Title: "Existing Article",
			Slug:  "existing-slug",
		}
		existing := &article.Article{ID: uuid.New(), Slug: req.Slug}
		mockStore.On("FindBySlug", ctx, req.Slug).Return(existing, nil).Once()

		result, err := svc.Create(ctx, req)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, core.ErrAlreadyExists)
		mockStore.AssertExpectations(t)
	})

	t.Run("creates article with tags", func(t *testing.T) {
		req := article.CreateRequest{
			Title:   "Tagged Article",
			Slug:    "tagged-article",
			Tags:    []string{"go", "backend"},
			IsDraft: true,
		}
		tagIDs := []int64{1, 2}
		mockStore.On("FindBySlug", ctx, req.Slug).Return(nil, core.ErrNotFound).Once()
		mockTagStore.On("EnsureExists", ctx, req.Tags).Return(tagIDs, nil).Once()
		mockStore.On("Save", ctx, mock.AnythingOfType("*article.Article")).Return(nil).Once()

		result, err := svc.Create(ctx, req)

		assert.NoError(t, err)
		assert.Equal(t, tagIDs, result.TagIDs)
		mockStore.AssertExpectations(t)
		mockTagStore.AssertExpectations(t)
	})
}

func TestService_Update(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockArticleStore)
	mockTagStore := new(mocks.MockTagStore)
	svc := article.NewService(mockStore, mockTagStore)

	t.Run("updates article successfully", func(t *testing.T) {
		articleID := uuid.New()
		existing := &article.Article{
			ID:      articleID,
			Title:   "Old Title",
			Content: "Old Content",
			Slug:    "old-slug",
		}
		newTitle := "New Title"
		req := article.UpdateRequest{
			Title: &newTitle,
		}
		mockStore.On("FindByID", ctx, articleID).Return(existing, nil).Once()
		mockStore.On("Save", ctx, existing).Return(nil).Once()

		result, err := svc.Update(ctx, articleID, req)

		assert.NoError(t, err)
		assert.Equal(t, newTitle, result.Title)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error when article not found", func(t *testing.T) {
		articleID := uuid.New()
		req := article.UpdateRequest{}
		mockStore.On("FindByID", ctx, articleID).Return(nil, core.ErrNotFound).Once()

		result, err := svc.Update(ctx, articleID, req)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, core.ErrNotFound)
		mockStore.AssertExpectations(t)
	})
}

func TestService_Delete(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockArticleStore)
	mockTagStore := new(mocks.MockTagStore)
	svc := article.NewService(mockStore, mockTagStore)

	t.Run("deletes article successfully", func(t *testing.T) {
		articleID := uuid.New()
		mockStore.On("Delete", ctx, articleID).Return(nil).Once()

		err := svc.Delete(ctx, articleID)

		assert.NoError(t, err)
		mockStore.AssertExpectations(t)
	})
}

func TestService_Search(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockArticleStore)
	mockTagStore := new(mocks.MockTagStore)
	svc := article.NewService(mockStore, mockTagStore)

	t.Run("returns search results", func(t *testing.T) {
		articles := []article.Article{
			{ID: uuid.New(), Title: "Match 1"},
		}
		opts := article.SearchOptions{Query: "test", Page: 1, PerPage: 10, IsDraft: nil}
		mockStore.On("Search", ctx, opts).Return(articles, int64(1), nil).Once()

		result, err := svc.Search(ctx, "test", 1, 10, nil)

		assert.NoError(t, err)
		assert.Equal(t, articles, result.Articles)
		mockStore.AssertExpectations(t)
	})
}
