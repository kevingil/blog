package project_test

import (
	"context"
	"testing"
	"time"

	"backend/pkg/core"
	"backend/pkg/core/project"
	"backend/pkg/types"
	"backend/testutil/mocks"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestService_GetByID(t *testing.T) {
	ctx := context.Background()

	t.Run("returns project when found", func(t *testing.T) {
		mockProjectStore := new(mocks.MockProjectRepository)
		mockTagStore := new(mocks.MockTagRepository)
		svc := project.NewService(mockProjectStore, mockTagStore)

		projectID := uuid.New()
		testProject := &types.Project{
			ID:          projectID,
			Title:       "Test Project",
			Description: "Test description",
			Content:     "Test content",
			TagIDs:      pq.Int64Array{1, 2},
			ImageURL:    "https://example.com/image.png",
			URL:         "https://example.com",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		mockProjectStore.On("FindByID", ctx, projectID).Return(testProject, nil).Once()

		result, err := svc.GetByID(ctx, projectID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Test Project", result.Title)
		assert.Equal(t, "Test description", result.Description)
		mockProjectStore.AssertExpectations(t)
	})

	t.Run("returns error when project not found", func(t *testing.T) {
		mockProjectStore := new(mocks.MockProjectRepository)
		mockTagStore := new(mocks.MockTagRepository)
		svc := project.NewService(mockProjectStore, mockTagStore)

		projectID := uuid.New()
		mockProjectStore.On("FindByID", ctx, projectID).Return(nil, core.ErrNotFound).Once()

		result, err := svc.GetByID(ctx, projectID)

		assert.ErrorIs(t, err, core.ErrNotFound)
		assert.Nil(t, result)
		mockProjectStore.AssertExpectations(t)
	})
}

func TestService_GetDetail(t *testing.T) {
	ctx := context.Background()

	t.Run("returns project with resolved tag names", func(t *testing.T) {
		mockProjectStore := new(mocks.MockProjectRepository)
		mockTagStore := new(mocks.MockTagRepository)
		svc := project.NewService(mockProjectStore, mockTagStore)

		projectID := uuid.New()
		testProject := &types.Project{
			ID:          projectID,
			Title:       "Test Project",
			Description: "Test description",
			Content:     "Test content",
			TagIDs:      pq.Int64Array{1, 2},
			ImageURL:    "https://example.com/image.png",
			URL:         "https://example.com",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		testTags := []types.Tag{
			{ID: 1, Name: "golang"},
			{ID: 2, Name: "testing"},
		}

		mockProjectStore.On("FindByID", ctx, projectID).Return(testProject, nil).Once()
		mockTagStore.On("FindByIDs", ctx, []int64{1, 2}).Return(testTags, nil).Once()

		result, err := svc.GetDetail(ctx, projectID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Test Project", result.Project.Title)
		assert.Len(t, result.Tags, 2)
		assert.Contains(t, result.Tags, "golang")
		assert.Contains(t, result.Tags, "testing")
		mockProjectStore.AssertExpectations(t)
		mockTagStore.AssertExpectations(t)
	})

	t.Run("returns error when project not found", func(t *testing.T) {
		mockProjectStore := new(mocks.MockProjectRepository)
		mockTagStore := new(mocks.MockTagRepository)
		svc := project.NewService(mockProjectStore, mockTagStore)

		projectID := uuid.New()
		mockProjectStore.On("FindByID", ctx, projectID).Return(nil, core.ErrNotFound).Once()

		result, err := svc.GetDetail(ctx, projectID)

		assert.ErrorIs(t, err, core.ErrNotFound)
		assert.Nil(t, result)
		mockProjectStore.AssertExpectations(t)
	})

	t.Run("returns project with empty tags when no tags exist", func(t *testing.T) {
		mockProjectStore := new(mocks.MockProjectRepository)
		mockTagStore := new(mocks.MockTagRepository)
		svc := project.NewService(mockProjectStore, mockTagStore)

		projectID := uuid.New()
		testProject := &types.Project{
			ID:          projectID,
			Title:       "Test Project",
			Description: "Test description",
			TagIDs:      pq.Int64Array{},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		mockProjectStore.On("FindByID", ctx, projectID).Return(testProject, nil).Once()

		result, err := svc.GetDetail(ctx, projectID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.Tags)
		mockProjectStore.AssertExpectations(t)
	})
}

func TestService_List(t *testing.T) {
	ctx := context.Background()

	t.Run("returns paginated projects", func(t *testing.T) {
		mockProjectStore := new(mocks.MockProjectRepository)
		mockTagStore := new(mocks.MockTagRepository)
		svc := project.NewService(mockProjectStore, mockTagStore)

		projects := []types.Project{
			{
				ID:          uuid.New(),
				Title:       "Project 1",
				Description: "Description 1",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			{
				ID:          uuid.New(),
				Title:       "Project 2",
				Description: "Description 2",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
		}

		opts := types.ProjectListOptions{
			Page:    1,
			PerPage: 10,
		}
		mockProjectStore.On("List", ctx, opts).Return(projects, int64(2), nil).Once()

		result, err := svc.List(ctx, 1, 10)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Projects, 2)
		assert.Equal(t, int64(2), result.Total)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 10, result.PerPage)
		assert.Equal(t, 1, result.TotalPages)
		mockProjectStore.AssertExpectations(t)
	})

	t.Run("uses default pagination when invalid values provided", func(t *testing.T) {
		mockProjectStore := new(mocks.MockProjectRepository)
		mockTagStore := new(mocks.MockTagRepository)
		svc := project.NewService(mockProjectStore, mockTagStore)

		opts := types.ProjectListOptions{
			Page:    1,
			PerPage: 20,
		}
		mockProjectStore.On("List", ctx, opts).Return([]types.Project{}, int64(0), nil).Once()

		result, err := svc.List(ctx, 0, 0)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 20, result.PerPage)
		mockProjectStore.AssertExpectations(t)
	})

	t.Run("calculates total pages correctly", func(t *testing.T) {
		mockProjectStore := new(mocks.MockProjectRepository)
		mockTagStore := new(mocks.MockTagRepository)
		svc := project.NewService(mockProjectStore, mockTagStore)

		opts := types.ProjectListOptions{
			Page:    1,
			PerPage: 10,
		}
		mockProjectStore.On("List", ctx, opts).Return([]types.Project{}, int64(25), nil).Once()

		result, err := svc.List(ctx, 1, 10)

		assert.NoError(t, err)
		assert.Equal(t, 3, result.TotalPages) // 25 items / 10 per page = 3 pages
		mockProjectStore.AssertExpectations(t)
	})
}

func TestService_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("creates project successfully", func(t *testing.T) {
		mockProjectStore := new(mocks.MockProjectRepository)
		mockTagStore := new(mocks.MockTagRepository)
		svc := project.NewService(mockProjectStore, mockTagStore)

		req := project.CreateRequest{
			Title:       "New Project",
			Description: "This is the description of the new project",
			Content:     "Project content here",
			Tags:        []string{"golang", "testing"},
			ImageURL:    "https://example.com/image.png",
			URL:         "https://example.com/project",
		}

		mockTagStore.On("EnsureExists", ctx, []string{"golang", "testing"}).Return([]int64{1, 2}, nil).Once()
		mockProjectStore.On("Save", ctx, mock.AnythingOfType("*types.Project")).Return(nil).Once()

		result, err := svc.Create(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "New Project", result.Title)
		assert.Equal(t, "This is the description of the new project", result.Description)
		assert.Equal(t, "Project content here", result.Content)
		assert.Equal(t, "https://example.com/image.png", result.ImageURL)
		assert.Equal(t, "https://example.com/project", result.URL)
		mockProjectStore.AssertExpectations(t)
		mockTagStore.AssertExpectations(t)
	})

	t.Run("creates project without tags", func(t *testing.T) {
		mockProjectStore := new(mocks.MockProjectRepository)
		mockTagStore := new(mocks.MockTagRepository)
		svc := project.NewService(mockProjectStore, mockTagStore)

		req := project.CreateRequest{
			Title:       "New Project",
			Description: "This is the description",
			Content:     "Content",
			Tags:        []string{},
		}

		mockTagStore.On("EnsureExists", ctx, []string{}).Return([]int64{}, nil).Once()
		mockProjectStore.On("Save", ctx, mock.AnythingOfType("*types.Project")).Return(nil).Once()

		result, err := svc.Create(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		mockProjectStore.AssertExpectations(t)
		mockTagStore.AssertExpectations(t)
	})

	t.Run("returns validation error when title is empty", func(t *testing.T) {
		mockProjectStore := new(mocks.MockProjectRepository)
		mockTagStore := new(mocks.MockTagRepository)
		svc := project.NewService(mockProjectStore, mockTagStore)

		req := project.CreateRequest{
			Title:       "",
			Description: "Valid description",
		}

		result, err := svc.Create(ctx, req)

		assert.ErrorIs(t, err, core.ErrValidation)
		assert.Nil(t, result)
	})

	t.Run("returns validation error when description is empty", func(t *testing.T) {
		mockProjectStore := new(mocks.MockProjectRepository)
		mockTagStore := new(mocks.MockTagRepository)
		svc := project.NewService(mockProjectStore, mockTagStore)

		req := project.CreateRequest{
			Title:       "Valid title",
			Description: "",
		}

		result, err := svc.Create(ctx, req)

		assert.ErrorIs(t, err, core.ErrValidation)
		assert.Nil(t, result)
	})
}

func TestService_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("updates project successfully", func(t *testing.T) {
		mockProjectStore := new(mocks.MockProjectRepository)
		mockTagStore := new(mocks.MockTagRepository)
		svc := project.NewService(mockProjectStore, mockTagStore)

		projectID := uuid.New()
		existingProject := &types.Project{
			ID:          projectID,
			Title:       "Original Title",
			Description: "Original description",
			Content:     "Original content",
			TagIDs:      pq.Int64Array{1},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		newTitle := "Updated Title"
		newDescription := "Updated description"
		newContent := "Updated content"
		newTags := []string{"newtag"}
		newImageURL := "https://example.com/new-image.png"
		newURL := "https://example.com/new-url"

		req := project.UpdateRequest{
			Title:       &newTitle,
			Description: &newDescription,
			Content:     &newContent,
			Tags:        &newTags,
			ImageURL:    &newImageURL,
			URL:         &newURL,
		}

		mockProjectStore.On("FindByID", ctx, projectID).Return(existingProject, nil).Once()
		mockTagStore.On("EnsureExists", ctx, []string{"newtag"}).Return([]int64{2}, nil).Once()
		mockProjectStore.On("Update", ctx, mock.AnythingOfType("*types.Project")).Return(nil).Once()

		result, err := svc.Update(ctx, projectID, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Updated Title", result.Title)
		assert.Equal(t, "Updated description", result.Description)
		assert.Equal(t, "Updated content", result.Content)
		assert.Equal(t, "https://example.com/new-image.png", result.ImageURL)
		assert.Equal(t, "https://example.com/new-url", result.URL)
		mockProjectStore.AssertExpectations(t)
		mockTagStore.AssertExpectations(t)
	})

	t.Run("updates only provided fields", func(t *testing.T) {
		mockProjectStore := new(mocks.MockProjectRepository)
		mockTagStore := new(mocks.MockTagRepository)
		svc := project.NewService(mockProjectStore, mockTagStore)

		projectID := uuid.New()
		existingProject := &types.Project{
			ID:          projectID,
			Title:       "Original Title",
			Description: "Original description",
			Content:     "Original content",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		newTitle := "Updated Title Only"
		req := project.UpdateRequest{
			Title: &newTitle,
		}

		mockProjectStore.On("FindByID", ctx, projectID).Return(existingProject, nil).Once()
		mockProjectStore.On("Update", ctx, mock.AnythingOfType("*types.Project")).Return(nil).Once()

		result, err := svc.Update(ctx, projectID, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Updated Title Only", result.Title)
		assert.Equal(t, "Original description", result.Description)
		assert.Equal(t, "Original content", result.Content)
		mockProjectStore.AssertExpectations(t)
	})

	t.Run("returns error when project not found", func(t *testing.T) {
		mockProjectStore := new(mocks.MockProjectRepository)
		mockTagStore := new(mocks.MockTagRepository)
		svc := project.NewService(mockProjectStore, mockTagStore)

		projectID := uuid.New()
		newTitle := "Updated Title"
		req := project.UpdateRequest{
			Title: &newTitle,
		}

		mockProjectStore.On("FindByID", ctx, projectID).Return(nil, core.ErrNotFound).Once()

		result, err := svc.Update(ctx, projectID, req)

		assert.ErrorIs(t, err, core.ErrNotFound)
		assert.Nil(t, result)
		mockProjectStore.AssertExpectations(t)
	})
}

func TestService_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("deletes project successfully", func(t *testing.T) {
		mockProjectStore := new(mocks.MockProjectRepository)
		mockTagStore := new(mocks.MockTagRepository)
		svc := project.NewService(mockProjectStore, mockTagStore)

		projectID := uuid.New()
		mockProjectStore.On("Delete", ctx, projectID).Return(nil).Once()

		err := svc.Delete(ctx, projectID)

		assert.NoError(t, err)
		mockProjectStore.AssertExpectations(t)
	})

	t.Run("returns error when project not found", func(t *testing.T) {
		mockProjectStore := new(mocks.MockProjectRepository)
		mockTagStore := new(mocks.MockTagRepository)
		svc := project.NewService(mockProjectStore, mockTagStore)

		projectID := uuid.New()
		mockProjectStore.On("Delete", ctx, projectID).Return(core.ErrNotFound).Once()

		err := svc.Delete(ctx, projectID)

		assert.ErrorIs(t, err, core.ErrNotFound)
		mockProjectStore.AssertExpectations(t)
	})
}
