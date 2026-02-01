package article_test

import (
	"context"
	"testing"
	"time"

	"backend/pkg/core"
	"backend/pkg/core/article"
	"backend/pkg/types"
	"backend/testutil/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestService_GetByID(t *testing.T) {
	ctx := context.Background()

	t.Run("returns article with metadata when found", func(t *testing.T) {
		mockArticleStore := new(mocks.MockArticleRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		mockTagStore := new(mocks.MockTagRepository)
		svc := article.NewService(mockArticleStore, mockAccountStore, mockTagStore)

		articleID := uuid.New()
		authorID := uuid.New()
		testArticle := &types.Article{
			ID:           articleID,
			DraftTitle:   "Test Article",
			DraftContent: "Test content",
			Slug:         "test-article",
			AuthorID:     authorID,
			TagIDs:       []int64{1, 2},
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		testAccount := &types.Account{
			ID:   authorID,
			Name: "Test Author",
		}
		testTags := []types.Tag{
			{ID: 1, Name: "golang"},
			{ID: 2, Name: "testing"},
		}

		mockArticleStore.On("FindByID", ctx, articleID).Return(testArticle, nil).Once()
		mockAccountStore.On("FindByID", ctx, authorID).Return(testAccount, nil).Once()
		mockTagStore.On("FindByIDs", ctx, []int64{1, 2}).Return(testTags, nil).Once()

		result, err := svc.GetByID(ctx, articleID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Test Article", result.Article.DraftTitle)
		assert.Equal(t, "Test Author", result.Author.Name)
		assert.Len(t, result.Tags, 2)
		mockArticleStore.AssertExpectations(t)
		mockAccountStore.AssertExpectations(t)
		mockTagStore.AssertExpectations(t)
	})

	t.Run("returns error when article not found", func(t *testing.T) {
		mockArticleStore := new(mocks.MockArticleRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		mockTagStore := new(mocks.MockTagRepository)
		svc := article.NewService(mockArticleStore, mockAccountStore, mockTagStore)

		articleID := uuid.New()
		mockArticleStore.On("FindByID", ctx, articleID).Return(nil, core.ErrNotFound).Once()

		result, err := svc.GetByID(ctx, articleID)

		assert.ErrorIs(t, err, core.ErrNotFound)
		assert.Nil(t, result)
		mockArticleStore.AssertExpectations(t)
	})
}

func TestService_GetBySlug(t *testing.T) {
	ctx := context.Background()

	t.Run("returns article data when found", func(t *testing.T) {
		mockArticleStore := new(mocks.MockArticleRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		mockTagStore := new(mocks.MockTagRepository)
		svc := article.NewService(mockArticleStore, mockAccountStore, mockTagStore)

		authorID := uuid.New()
		testArticle := &types.Article{
			ID:           uuid.New(),
			DraftTitle:   "Test Article",
			DraftContent: "Test content",
			Slug:         "test-article",
			AuthorID:     authorID,
			TagIDs:       []int64{1},
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		testAccount := &types.Account{
			ID:   authorID,
			Name: "Author Name",
		}
		testTags := []types.Tag{
			{ID: 1, Name: "golang"},
		}

		mockArticleStore.On("FindBySlug", ctx, "test-article").Return(testArticle, nil).Once()
		mockAccountStore.On("FindByID", ctx, authorID).Return(testAccount, nil).Once()
		mockTagStore.On("FindByIDs", ctx, []int64{1}).Return(testTags, nil).Once()

		result, err := svc.GetBySlug(ctx, "test-article")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Test Article", result.Article.DraftTitle)
		assert.Equal(t, "Author Name", result.Author.Name)
		mockArticleStore.AssertExpectations(t)
	})

	t.Run("returns error when slug not found", func(t *testing.T) {
		mockArticleStore := new(mocks.MockArticleRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		mockTagStore := new(mocks.MockTagRepository)
		svc := article.NewService(mockArticleStore, mockAccountStore, mockTagStore)

		mockArticleStore.On("FindBySlug", ctx, "nonexistent").Return(nil, core.ErrNotFound).Once()

		result, err := svc.GetBySlug(ctx, "nonexistent")

		assert.ErrorIs(t, err, core.ErrNotFound)
		assert.Nil(t, result)
		mockArticleStore.AssertExpectations(t)
	})
}

func TestService_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("creates article successfully", func(t *testing.T) {
		mockArticleStore := new(mocks.MockArticleRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		mockTagStore := new(mocks.MockTagRepository)
		svc := article.NewService(mockArticleStore, mockAccountStore, mockTagStore)

		authorID := uuid.New()
		req := article.CreateRequest{
			Title:    "New Article",
			Content:  "This is the content of the new article",
			Tags:     []string{"golang", "testing"},
			AuthorID: authorID,
		}

		mockTagStore.On("EnsureExists", ctx, []string{"golang", "testing"}).Return([]int64{1, 2}, nil).Once()
		mockArticleStore.On("SlugExists", ctx, "new-article", (*uuid.UUID)(nil)).Return(false, nil).Once()
		mockArticleStore.On("Save", ctx, mock.AnythingOfType("*types.Article")).Return(nil).Once()

		// Mock for GetByID call after create
		mockArticleStore.On("FindByID", ctx, mock.AnythingOfType("uuid.UUID")).Return(&types.Article{
			ID:           uuid.New(),
			DraftTitle:   "New Article",
			DraftContent: "This is the content of the new article",
			Slug:         "new-article",
			AuthorID:     authorID,
			TagIDs:       []int64{1, 2},
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}, nil).Once()
		mockAccountStore.On("FindByID", ctx, authorID).Return(&types.Account{ID: authorID, Name: "Author"}, nil).Once()
		mockTagStore.On("FindByIDs", ctx, []int64{1, 2}).Return([]types.Tag{{ID: 1, Name: "golang"}, {ID: 2, Name: "testing"}}, nil).Once()

		result, err := svc.Create(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "New Article", result.Article.DraftTitle)
		mockArticleStore.AssertExpectations(t)
		mockTagStore.AssertExpectations(t)
	})

	t.Run("creates article with unique slug when slug exists", func(t *testing.T) {
		mockArticleStore := new(mocks.MockArticleRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		mockTagStore := new(mocks.MockTagRepository)
		svc := article.NewService(mockArticleStore, mockAccountStore, mockTagStore)

		authorID := uuid.New()
		req := article.CreateRequest{
			Title:    "Duplicate Title",
			Content:  "This is the content",
			Tags:     []string{},
			AuthorID: authorID,
		}

		mockTagStore.On("EnsureExists", ctx, []string{}).Return([]int64{}, nil).Once()
		// First slug check returns true (slug exists)
		mockArticleStore.On("SlugExists", ctx, "duplicate-title", (*uuid.UUID)(nil)).Return(true, nil).Once()
		mockArticleStore.On("Save", ctx, mock.AnythingOfType("*types.Article")).Return(nil).Once()

		// Mock for GetByID call
		mockArticleStore.On("FindByID", ctx, mock.AnythingOfType("uuid.UUID")).Return(&types.Article{
			ID:           uuid.New(),
			DraftTitle:   "Duplicate Title",
			DraftContent: "This is the content",
			Slug:         "duplicate-title-abc12345",
			AuthorID:     authorID,
			TagIDs:       []int64{},
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}, nil).Once()
		mockAccountStore.On("FindByID", ctx, authorID).Return(&types.Account{ID: authorID, Name: "Author"}, nil).Once()
		mockTagStore.On("FindByIDs", ctx, []int64{}).Return([]types.Tag{}, nil).Once()

		result, err := svc.Create(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		// Slug should be different from base slug due to uniqueness
		assert.NotEqual(t, "duplicate-title", result.Article.Slug)
		mockArticleStore.AssertExpectations(t)
	})
}

func TestService_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("deletes article successfully", func(t *testing.T) {
		mockArticleStore := new(mocks.MockArticleRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		mockTagStore := new(mocks.MockTagRepository)
		svc := article.NewService(mockArticleStore, mockAccountStore, mockTagStore)

		articleID := uuid.New()
		mockArticleStore.On("Delete", ctx, articleID).Return(nil).Once()

		err := svc.Delete(ctx, articleID)

		assert.NoError(t, err)
		mockArticleStore.AssertExpectations(t)
	})

	t.Run("returns error when article not found", func(t *testing.T) {
		mockArticleStore := new(mocks.MockArticleRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		mockTagStore := new(mocks.MockTagRepository)
		svc := article.NewService(mockArticleStore, mockAccountStore, mockTagStore)

		articleID := uuid.New()
		mockArticleStore.On("Delete", ctx, articleID).Return(core.ErrNotFound).Once()

		err := svc.Delete(ctx, articleID)

		assert.ErrorIs(t, err, core.ErrNotFound)
		mockArticleStore.AssertExpectations(t)
	})
}

func TestService_Publish(t *testing.T) {
	ctx := context.Background()

	t.Run("publishes article successfully", func(t *testing.T) {
		mockArticleStore := new(mocks.MockArticleRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		mockTagStore := new(mocks.MockTagRepository)
		svc := article.NewService(mockArticleStore, mockAccountStore, mockTagStore)

		articleID := uuid.New()
		authorID := uuid.New()
		testArticle := &types.Article{
			ID:           articleID,
			DraftTitle:   "Draft Article",
			DraftContent: "Draft content",
			Slug:         "draft-article",
			AuthorID:     authorID,
			TagIDs:       []int64{},
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		mockArticleStore.On("FindByID", ctx, articleID).Return(testArticle, nil).Once()
		mockArticleStore.On("Publish", ctx, testArticle).Return(nil).Once()
		// Mock for GetByID after publish
		mockArticleStore.On("FindByID", ctx, articleID).Return(testArticle, nil).Once()
		mockAccountStore.On("FindByID", ctx, authorID).Return(&types.Account{ID: authorID, Name: "Author"}, nil).Once()
		mockTagStore.On("FindByIDs", ctx, []int64{}).Return([]types.Tag{}, nil).Once()

		result, err := svc.Publish(ctx, articleID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		mockArticleStore.AssertExpectations(t)
	})
}

func TestService_Unpublish(t *testing.T) {
	ctx := context.Background()

	t.Run("unpublishes article successfully", func(t *testing.T) {
		mockArticleStore := new(mocks.MockArticleRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		mockTagStore := new(mocks.MockTagRepository)
		svc := article.NewService(mockArticleStore, mockAccountStore, mockTagStore)

		articleID := uuid.New()
		authorID := uuid.New()
		now := time.Now()
		title := "Published Article"
		testArticle := &types.Article{
			ID:             articleID,
			DraftTitle:     "Published Article",
			DraftContent:   "Published content",
			PublishedTitle: &title,
			PublishedAt:    &now,
			Slug:           "published-article",
			AuthorID:       authorID,
			TagIDs:         []int64{},
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		mockArticleStore.On("FindByID", ctx, articleID).Return(testArticle, nil).Once()
		mockArticleStore.On("Unpublish", ctx, testArticle).Return(nil).Once()
		// Mock for GetByID after unpublish
		mockArticleStore.On("FindByID", ctx, articleID).Return(testArticle, nil).Once()
		mockAccountStore.On("FindByID", ctx, authorID).Return(&types.Account{ID: authorID, Name: "Author"}, nil).Once()
		mockTagStore.On("FindByIDs", ctx, []int64{}).Return([]types.Tag{}, nil).Once()

		result, err := svc.Unpublish(ctx, articleID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		mockArticleStore.AssertExpectations(t)
	})

	t.Run("returns error when article is not published", func(t *testing.T) {
		mockArticleStore := new(mocks.MockArticleRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		mockTagStore := new(mocks.MockTagRepository)
		svc := article.NewService(mockArticleStore, mockAccountStore, mockTagStore)

		articleID := uuid.New()
		testArticle := &types.Article{
			ID:          articleID,
			DraftTitle:  "Draft Article",
			PublishedAt: nil, // Not published
		}

		mockArticleStore.On("FindByID", ctx, articleID).Return(testArticle, nil).Once()

		result, err := svc.Unpublish(ctx, articleID)

		assert.ErrorIs(t, err, core.ErrValidation)
		assert.Nil(t, result)
		mockArticleStore.AssertExpectations(t)
	})
}

func TestService_GetPopularTags(t *testing.T) {
	ctx := context.Background()

	t.Run("returns popular tag names", func(t *testing.T) {
		mockArticleStore := new(mocks.MockArticleRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		mockTagStore := new(mocks.MockTagRepository)
		svc := article.NewService(mockArticleStore, mockAccountStore, mockTagStore)

		mockArticleStore.On("GetPopularTags", ctx, 10).Return([]int64{1, 2, 3}, nil).Once()
		mockTagStore.On("FindByIDs", ctx, []int64{1, 2, 3}).Return([]types.Tag{
			{ID: 1, Name: "golang"},
			{ID: 2, Name: "testing"},
			{ID: 3, Name: "backend"},
		}, nil).Once()

		result, err := svc.GetPopularTags(ctx)

		assert.NoError(t, err)
		assert.Len(t, result, 3)
		assert.Contains(t, result, "golang")
		assert.Contains(t, result, "testing")
		assert.Contains(t, result, "backend")
		mockArticleStore.AssertExpectations(t)
		mockTagStore.AssertExpectations(t)
	})
}
