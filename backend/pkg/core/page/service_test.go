package page_test

import (
	"context"
	"testing"
	"time"

	"backend/pkg/core"
	"backend/pkg/core/page"
	"backend/pkg/types"
	"backend/testutil/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestService_GetByID(t *testing.T) {
	ctx := context.Background()

	t.Run("returns page when found", func(t *testing.T) {
		mockStore := new(mocks.MockPageStore)
		svc := page.NewService(mockStore)

		pageID := uuid.New()
		testPage := &types.Page{
			ID:          pageID,
			Slug:        "about-us",
			Title:       "About Us",
			Content:     "This is the about us page content.",
			Description: "Learn more about our company",
			IsPublished: true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		mockStore.On("FindByID", ctx, pageID).Return(testPage, nil).Once()

		result, err := svc.GetByID(ctx, pageID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "About Us", result.Title)
		assert.Equal(t, "about-us", result.Slug)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error when page not found", func(t *testing.T) {
		mockStore := new(mocks.MockPageStore)
		svc := page.NewService(mockStore)

		pageID := uuid.New()
		mockStore.On("FindByID", ctx, pageID).Return(nil, core.ErrNotFound).Once()

		result, err := svc.GetByID(ctx, pageID)

		assert.ErrorIs(t, err, core.ErrNotFound)
		assert.Nil(t, result)
		mockStore.AssertExpectations(t)
	})
}

func TestService_GetBySlug(t *testing.T) {
	ctx := context.Background()

	t.Run("returns page when found", func(t *testing.T) {
		mockStore := new(mocks.MockPageStore)
		svc := page.NewService(mockStore)

		testPage := &types.Page{
			ID:          uuid.New(),
			Slug:        "contact",
			Title:       "Contact Us",
			Content:     "Get in touch with us.",
			Description: "Contact information",
			IsPublished: true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		mockStore.On("FindBySlug", ctx, "contact").Return(testPage, nil).Once()

		result, err := svc.GetBySlug(ctx, "contact")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Contact Us", result.Title)
		assert.Equal(t, "contact", result.Slug)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error when slug not found", func(t *testing.T) {
		mockStore := new(mocks.MockPageStore)
		svc := page.NewService(mockStore)

		mockStore.On("FindBySlug", ctx, "nonexistent").Return(nil, core.ErrNotFound).Once()

		result, err := svc.GetBySlug(ctx, "nonexistent")

		assert.ErrorIs(t, err, core.ErrNotFound)
		assert.Nil(t, result)
		mockStore.AssertExpectations(t)
	})
}

func TestService_List(t *testing.T) {
	ctx := context.Background()

	t.Run("returns pages with pagination", func(t *testing.T) {
		mockStore := new(mocks.MockPageStore)
		svc := page.NewService(mockStore)

		testPages := []types.Page{
			{
				ID:          uuid.New(),
				Slug:        "about",
				Title:       "About",
				Content:     "About page content",
				IsPublished: true,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			{
				ID:          uuid.New(),
				Slug:        "contact",
				Title:       "Contact",
				Content:     "Contact page content",
				IsPublished: true,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
		}

		expectedOpts := types.PageListOptions{
			Page:        1,
			PerPage:     10,
			IsPublished: nil,
		}
		mockStore.On("List", ctx, expectedOpts).Return(testPages, int64(2), nil).Once()

		result, err := svc.List(ctx, 1, 10, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Pages, 2)
		assert.Equal(t, int64(2), result.Total)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 10, result.PerPage)
		assert.Equal(t, 1, result.TotalPages)
		mockStore.AssertExpectations(t)
	})

	t.Run("applies default pagination values", func(t *testing.T) {
		mockStore := new(mocks.MockPageStore)
		svc := page.NewService(mockStore)

		expectedOpts := types.PageListOptions{
			Page:        1,
			PerPage:     20,
			IsPublished: nil,
		}
		mockStore.On("List", ctx, expectedOpts).Return([]types.Page{}, int64(0), nil).Once()

		result, err := svc.List(ctx, 0, 0, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 20, result.PerPage)
		mockStore.AssertExpectations(t)
	})

	t.Run("filters by published status", func(t *testing.T) {
		mockStore := new(mocks.MockPageStore)
		svc := page.NewService(mockStore)

		isPublished := true
		expectedOpts := types.PageListOptions{
			Page:        1,
			PerPage:     20,
			IsPublished: &isPublished,
		}
		mockStore.On("List", ctx, expectedOpts).Return([]types.Page{}, int64(0), nil).Once()

		result, err := svc.List(ctx, 1, 20, &isPublished)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		mockStore.AssertExpectations(t)
	})
}

func TestService_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("creates page successfully", func(t *testing.T) {
		mockStore := new(mocks.MockPageStore)
		svc := page.NewService(mockStore)

		req := page.CreateRequest{
			Slug:        "new-page",
			Title:       "New Page",
			Content:     "This is the content of the new page.",
			Description: "A brand new page",
			IsPublished: true,
		}

		// Slug check returns not found (slug is available)
		mockStore.On("FindBySlug", ctx, "new-page").Return(nil, core.ErrNotFound).Once()
		mockStore.On("Save", ctx, mock.AnythingOfType("*types.Page")).Return(nil).Once()

		result, err := svc.Create(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "New Page", result.Title)
		assert.Equal(t, "new-page", result.Slug)
		assert.Equal(t, "This is the content of the new page.", result.Content)
		assert.True(t, result.IsPublished)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error when slug already exists", func(t *testing.T) {
		mockStore := new(mocks.MockPageStore)
		svc := page.NewService(mockStore)

		req := page.CreateRequest{
			Slug:    "existing-page",
			Title:   "Existing Page",
			Content: "This content will not be saved.",
		}

		existingPage := &types.Page{
			ID:   uuid.New(),
			Slug: "existing-page",
		}
		mockStore.On("FindBySlug", ctx, "existing-page").Return(existingPage, nil).Once()

		result, err := svc.Create(ctx, req)

		assert.ErrorIs(t, err, core.ErrAlreadyExists)
		assert.Nil(t, result)
		mockStore.AssertExpectations(t)
	})
}

func TestService_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("updates page successfully", func(t *testing.T) {
		mockStore := new(mocks.MockPageStore)
		svc := page.NewService(mockStore)

		pageID := uuid.New()
		existingPage := &types.Page{
			ID:          pageID,
			Slug:        "about-us",
			Title:       "About Us",
			Content:     "Original content",
			Description: "Original description",
			IsPublished: false,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		newTitle := "Updated About Us"
		newContent := "Updated content here"
		isPublished := true
		req := page.UpdateRequest{
			Title:       &newTitle,
			Content:     &newContent,
			IsPublished: &isPublished,
		}

		mockStore.On("FindByID", ctx, pageID).Return(existingPage, nil).Once()
		mockStore.On("Save", ctx, mock.AnythingOfType("*types.Page")).Return(nil).Once()

		result, err := svc.Update(ctx, pageID, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Updated About Us", result.Title)
		assert.Equal(t, "Updated content here", result.Content)
		assert.True(t, result.IsPublished)
		mockStore.AssertExpectations(t)
	})

	t.Run("updates only provided fields", func(t *testing.T) {
		mockStore := new(mocks.MockPageStore)
		svc := page.NewService(mockStore)

		pageID := uuid.New()
		existingPage := &types.Page{
			ID:          pageID,
			Slug:        "contact",
			Title:       "Contact",
			Content:     "Original contact content",
			Description: "Contact description",
			IsPublished: true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		newDescription := "New contact description"
		req := page.UpdateRequest{
			Description: &newDescription,
		}

		mockStore.On("FindByID", ctx, pageID).Return(existingPage, nil).Once()
		mockStore.On("Save", ctx, mock.AnythingOfType("*types.Page")).Return(nil).Once()

		result, err := svc.Update(ctx, pageID, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Contact", result.Title) // unchanged
		assert.Equal(t, "Original contact content", result.Content) // unchanged
		assert.Equal(t, "New contact description", result.Description) // updated
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error when page not found", func(t *testing.T) {
		mockStore := new(mocks.MockPageStore)
		svc := page.NewService(mockStore)

		pageID := uuid.New()
		newTitle := "Updated Title"
		req := page.UpdateRequest{
			Title: &newTitle,
		}

		mockStore.On("FindByID", ctx, pageID).Return(nil, core.ErrNotFound).Once()

		result, err := svc.Update(ctx, pageID, req)

		assert.ErrorIs(t, err, core.ErrNotFound)
		assert.Nil(t, result)
		mockStore.AssertExpectations(t)
	})
}

func TestService_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("deletes page successfully", func(t *testing.T) {
		mockStore := new(mocks.MockPageStore)
		svc := page.NewService(mockStore)

		pageID := uuid.New()
		mockStore.On("Delete", ctx, pageID).Return(nil).Once()

		err := svc.Delete(ctx, pageID)

		assert.NoError(t, err)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error when page not found", func(t *testing.T) {
		mockStore := new(mocks.MockPageStore)
		svc := page.NewService(mockStore)

		pageID := uuid.New()
		mockStore.On("Delete", ctx, pageID).Return(core.ErrNotFound).Once()

		err := svc.Delete(ctx, pageID)

		assert.ErrorIs(t, err, core.ErrNotFound)
		mockStore.AssertExpectations(t)
	})
}
