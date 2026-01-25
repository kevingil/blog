package page_test

import (
	"context"
	"testing"

	"backend/pkg/core"
	"backend/pkg/core/page"
	"backend/testutil/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestService_GetByID(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockPageStore)
	svc := page.NewService(mockStore)

	t.Run("returns page when found", func(t *testing.T) {
		pageID := uuid.New()
		expectedPage := &page.Page{
			ID:    pageID,
			Slug:  "test-page",
			Title: "Test Page",
		}
		mockStore.On("FindByID", ctx, pageID).Return(expectedPage, nil).Once()

		result, err := svc.GetByID(ctx, pageID)

		assert.NoError(t, err)
		assert.Equal(t, expectedPage, result)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		pageID := uuid.New()
		mockStore.On("FindByID", ctx, pageID).Return(nil, core.ErrNotFound).Once()

		result, err := svc.GetByID(ctx, pageID)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, core.ErrNotFound)
		mockStore.AssertExpectations(t)
	})
}

func TestService_GetBySlug(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockPageStore)
	svc := page.NewService(mockStore)

	t.Run("returns page when found", func(t *testing.T) {
		slug := "test-page"
		expectedPage := &page.Page{
			ID:    uuid.New(),
			Slug:  slug,
			Title: "Test Page",
		}
		mockStore.On("FindBySlug", ctx, slug).Return(expectedPage, nil).Once()

		result, err := svc.GetBySlug(ctx, slug)

		assert.NoError(t, err)
		assert.Equal(t, expectedPage, result)
		mockStore.AssertExpectations(t)
	})
}

func TestService_List(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockPageStore)
	svc := page.NewService(mockStore)

	t.Run("returns paginated list", func(t *testing.T) {
		pages := []page.Page{
			{ID: uuid.New(), Slug: "page-1", Title: "Page 1"},
			{ID: uuid.New(), Slug: "page-2", Title: "Page 2"},
		}
		mockStore.On("List", ctx, page.ListOptions{Page: 1, PerPage: 10}).Return(pages, int64(2), nil).Once()

		result, err := svc.List(ctx, 1, 10)

		assert.NoError(t, err)
		assert.Equal(t, pages, result.Pages)
		assert.Equal(t, int64(2), result.Total)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 10, result.PerPage)
		assert.Equal(t, 1, result.TotalPages)
		mockStore.AssertExpectations(t)
	})

	t.Run("uses default values for invalid pagination", func(t *testing.T) {
		pages := []page.Page{}
		mockStore.On("List", ctx, page.ListOptions{Page: 1, PerPage: 20}).Return(pages, int64(0), nil).Once()

		result, err := svc.List(ctx, 0, 0)

		assert.NoError(t, err)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 20, result.PerPage)
		mockStore.AssertExpectations(t)
	})
}

func TestService_Create(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockPageStore)
	svc := page.NewService(mockStore)

	t.Run("creates page successfully", func(t *testing.T) {
		req := page.CreateRequest{
			Slug:        "new-page",
			Title:       "New Page",
			Content:     "Content",
			IsPublished: true,
		}
		mockStore.On("FindBySlug", ctx, req.Slug).Return(nil, core.ErrNotFound).Once()
		mockStore.On("Save", ctx, mock.AnythingOfType("*page.Page")).Return(nil).Once()

		result, err := svc.Create(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, req.Slug, result.Slug)
		assert.Equal(t, req.Title, result.Title)
		assert.Equal(t, req.Content, result.Content)
		assert.Equal(t, req.IsPublished, result.IsPublished)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error when slug already exists", func(t *testing.T) {
		req := page.CreateRequest{
			Slug:  "existing-page",
			Title: "Existing Page",
		}
		existingPage := &page.Page{ID: uuid.New(), Slug: req.Slug}
		mockStore.On("FindBySlug", ctx, req.Slug).Return(existingPage, nil).Once()

		result, err := svc.Create(ctx, req)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, core.ErrAlreadyExists)
		mockStore.AssertExpectations(t)
	})
}

func TestService_Update(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockPageStore)
	svc := page.NewService(mockStore)

	t.Run("updates page successfully", func(t *testing.T) {
		pageID := uuid.New()
		existingPage := &page.Page{
			ID:      pageID,
			Slug:    "test-page",
			Title:   "Old Title",
			Content: "Old Content",
		}
		newTitle := "New Title"
		newContent := "New Content"
		req := page.UpdateRequest{
			Title:   &newTitle,
			Content: &newContent,
		}
		mockStore.On("FindByID", ctx, pageID).Return(existingPage, nil).Once()
		mockStore.On("Save", ctx, existingPage).Return(nil).Once()

		result, err := svc.Update(ctx, pageID, req)

		assert.NoError(t, err)
		assert.Equal(t, newTitle, result.Title)
		assert.Equal(t, newContent, result.Content)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error when page not found", func(t *testing.T) {
		pageID := uuid.New()
		req := page.UpdateRequest{}
		mockStore.On("FindByID", ctx, pageID).Return(nil, core.ErrNotFound).Once()

		result, err := svc.Update(ctx, pageID, req)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, core.ErrNotFound)
		mockStore.AssertExpectations(t)
	})
}

func TestService_Delete(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockPageStore)
	svc := page.NewService(mockStore)

	t.Run("deletes page successfully", func(t *testing.T) {
		pageID := uuid.New()
		mockStore.On("Delete", ctx, pageID).Return(nil).Once()

		err := svc.Delete(ctx, pageID)

		assert.NoError(t, err)
		mockStore.AssertExpectations(t)
	})
}
