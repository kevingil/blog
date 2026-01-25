package project_test

import (
	"context"
	"testing"

	"backend/pkg/core"
	"backend/pkg/core/project"
	"backend/pkg/core/tag"
	"backend/testutil/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestService_GetByID(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockProjectStore)
	mockTagStore := new(mocks.MockTagStore)
	svc := project.NewService(mockStore, mockTagStore)

	t.Run("returns project when found", func(t *testing.T) {
		projectID := uuid.New()
		expected := &project.Project{ID: projectID, Title: "Test Project"}
		mockStore.On("FindByID", ctx, projectID).Return(expected, nil).Once()

		result, err := svc.GetByID(ctx, projectID)

		assert.NoError(t, err)
		assert.Equal(t, expected, result)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		projectID := uuid.New()
		mockStore.On("FindByID", ctx, projectID).Return(nil, core.ErrNotFound).Once()

		result, err := svc.GetByID(ctx, projectID)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, core.ErrNotFound)
		mockStore.AssertExpectations(t)
	})
}

func TestService_GetDetail(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockProjectStore)
	mockTagStore := new(mocks.MockTagStore)
	svc := project.NewService(mockStore, mockTagStore)

	t.Run("returns project with resolved tags", func(t *testing.T) {
		projectID := uuid.New()
		proj := &project.Project{ID: projectID, Title: "Test Project", TagIDs: []int64{1, 2}}
		tags := []tag.Tag{{ID: 1, Name: "go"}, {ID: 2, Name: "backend"}}

		mockStore.On("FindByID", ctx, projectID).Return(proj, nil).Once()
		mockTagStore.On("FindByIDs", ctx, []int64{1, 2}).Return(tags, nil).Once()

		result, err := svc.GetDetail(ctx, projectID)

		assert.NoError(t, err)
		assert.Equal(t, proj.Title, result.Project.Title)
		assert.Equal(t, []string{"go", "backend"}, result.Tags)
		mockStore.AssertExpectations(t)
		mockTagStore.AssertExpectations(t)
	})
}

func TestService_List(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockProjectStore)
	mockTagStore := new(mocks.MockTagStore)
	svc := project.NewService(mockStore, mockTagStore)

	t.Run("returns paginated list", func(t *testing.T) {
		projects := []project.Project{
			{ID: uuid.New(), Title: "Project 1"},
			{ID: uuid.New(), Title: "Project 2"},
		}
		opts := project.ListOptions{Page: 1, PerPage: 10}
		mockStore.On("List", ctx, opts).Return(projects, int64(2), nil).Once()

		result, err := svc.List(ctx, 1, 10)

		assert.NoError(t, err)
		assert.Equal(t, projects, result.Projects)
		assert.Equal(t, int64(2), result.Total)
		mockStore.AssertExpectations(t)
	})

	t.Run("uses default values for invalid pagination", func(t *testing.T) {
		projects := []project.Project{}
		opts := project.ListOptions{Page: 1, PerPage: 20}
		mockStore.On("List", ctx, opts).Return(projects, int64(0), nil).Once()

		result, err := svc.List(ctx, 0, 0)

		assert.NoError(t, err)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 20, result.PerPage)
		mockStore.AssertExpectations(t)
	})
}

func TestService_Create(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockProjectStore)
	mockTagStore := new(mocks.MockTagStore)
	svc := project.NewService(mockStore, mockTagStore)

	t.Run("creates project successfully", func(t *testing.T) {
		req := project.CreateRequest{
			Title:       "New Project",
			Description: "A great project",
			Content:     "Detailed content",
		}
		mockStore.On("Save", ctx, mock.AnythingOfType("*project.Project")).Return(nil).Once()

		result, err := svc.Create(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, req.Title, result.Title)
		assert.Equal(t, req.Description, result.Description)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error when title is empty", func(t *testing.T) {
		req := project.CreateRequest{
			Title:       "",
			Description: "Description",
		}

		result, err := svc.Create(ctx, req)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, core.ErrValidation)
	})

	t.Run("returns error when description is empty", func(t *testing.T) {
		req := project.CreateRequest{
			Title:       "Title",
			Description: "",
		}

		result, err := svc.Create(ctx, req)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, core.ErrValidation)
	})

	t.Run("creates project with tags", func(t *testing.T) {
		req := project.CreateRequest{
			Title:       "Tagged Project",
			Description: "Has tags",
			Tags:        []string{"go", "backend"},
		}
		tagIDs := []int64{1, 2}
		mockTagStore.On("EnsureExists", ctx, req.Tags).Return(tagIDs, nil).Once()
		mockStore.On("Save", ctx, mock.AnythingOfType("*project.Project")).Return(nil).Once()

		result, err := svc.Create(ctx, req)

		assert.NoError(t, err)
		assert.Equal(t, tagIDs, result.TagIDs)
		mockStore.AssertExpectations(t)
		mockTagStore.AssertExpectations(t)
	})
}

func TestService_Update(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockProjectStore)
	mockTagStore := new(mocks.MockTagStore)
	svc := project.NewService(mockStore, mockTagStore)

	t.Run("updates project successfully", func(t *testing.T) {
		projectID := uuid.New()
		existing := &project.Project{
			ID:          projectID,
			Title:       "Old Title",
			Description: "Old Description",
		}
		newTitle := "New Title"
		req := project.UpdateRequest{
			Title: &newTitle,
		}
		mockStore.On("FindByID", ctx, projectID).Return(existing, nil).Once()
		mockStore.On("Update", ctx, existing).Return(nil).Once()

		result, err := svc.Update(ctx, projectID, req)

		assert.NoError(t, err)
		assert.Equal(t, newTitle, result.Title)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error when project not found", func(t *testing.T) {
		projectID := uuid.New()
		req := project.UpdateRequest{}
		mockStore.On("FindByID", ctx, projectID).Return(nil, core.ErrNotFound).Once()

		result, err := svc.Update(ctx, projectID, req)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, core.ErrNotFound)
		mockStore.AssertExpectations(t)
	})
}

func TestService_Delete(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockProjectStore)
	mockTagStore := new(mocks.MockTagStore)
	svc := project.NewService(mockStore, mockTagStore)

	t.Run("deletes project successfully", func(t *testing.T) {
		projectID := uuid.New()
		mockStore.On("Delete", ctx, projectID).Return(nil).Once()

		err := svc.Delete(ctx, projectID)

		assert.NoError(t, err)
		mockStore.AssertExpectations(t)
	})
}
