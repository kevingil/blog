package tag_test

import (
	"context"
	"testing"

	"backend/pkg/core"
	"backend/pkg/core/tag"
	"backend/testutil/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestService_GetByID(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockTagStore)
	svc := tag.NewService(mockStore)

	t.Run("returns tag when found", func(t *testing.T) {
		expected := &tag.Tag{ID: 1, Name: "golang"}
		mockStore.On("FindByID", ctx, 1).Return(expected, nil).Once()

		result, err := svc.GetByID(ctx, 1)

		assert.NoError(t, err)
		assert.Equal(t, expected, result)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		mockStore.On("FindByID", ctx, 999).Return(nil, core.ErrNotFound).Once()

		result, err := svc.GetByID(ctx, 999)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, core.ErrNotFound)
		mockStore.AssertExpectations(t)
	})
}

func TestService_GetByName(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockTagStore)
	svc := tag.NewService(mockStore)

	t.Run("returns tag when found", func(t *testing.T) {
		expected := &tag.Tag{ID: 1, Name: "golang"}
		mockStore.On("FindByName", ctx, "golang").Return(expected, nil).Once()

		result, err := svc.GetByName(ctx, "golang")

		assert.NoError(t, err)
		assert.Equal(t, expected, result)
		mockStore.AssertExpectations(t)
	})
}

func TestService_GetByIDs(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockTagStore)
	svc := tag.NewService(mockStore)

	t.Run("returns tags for IDs", func(t *testing.T) {
		ids := []int64{1, 2, 3}
		tags := []tag.Tag{
			{ID: 1, Name: "go"},
			{ID: 2, Name: "backend"},
			{ID: 3, Name: "api"},
		}
		mockStore.On("FindByIDs", ctx, ids).Return(tags, nil).Once()

		result, err := svc.GetByIDs(ctx, ids)

		assert.NoError(t, err)
		assert.Equal(t, tags, result)
		mockStore.AssertExpectations(t)
	})
}

func TestService_EnsureExists(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockTagStore)
	svc := tag.NewService(mockStore)

	t.Run("creates tags if they don't exist", func(t *testing.T) {
		names := []string{"go", "backend"}
		ids := []int64{1, 2}
		mockStore.On("EnsureExists", ctx, names).Return(ids, nil).Once()

		result, err := svc.EnsureExists(ctx, names)

		assert.NoError(t, err)
		assert.Equal(t, ids, result)
		mockStore.AssertExpectations(t)
	})
}

func TestService_List(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockTagStore)
	svc := tag.NewService(mockStore)

	t.Run("returns all tags", func(t *testing.T) {
		tags := []tag.Tag{
			{ID: 1, Name: "go"},
			{ID: 2, Name: "backend"},
		}
		mockStore.On("List", ctx).Return(tags, nil).Once()

		result, err := svc.List(ctx)

		assert.NoError(t, err)
		assert.Equal(t, tags, result)
		mockStore.AssertExpectations(t)
	})
}

func TestService_Create(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockTagStore)
	svc := tag.NewService(mockStore)

	t.Run("creates tag successfully", func(t *testing.T) {
		mockStore.On("Save", ctx, mock.AnythingOfType("*types.Tag")).Return(nil).Once()

		result, err := svc.Create(ctx, "newtag")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "newtag", result.Name)
		mockStore.AssertExpectations(t)
	})
}

func TestService_Delete(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockTagStore)
	svc := tag.NewService(mockStore)

	t.Run("deletes tag successfully", func(t *testing.T) {
		mockStore.On("Delete", ctx, 1).Return(nil).Once()

		err := svc.Delete(ctx, 1)

		assert.NoError(t, err)
		mockStore.AssertExpectations(t)
	})
}

func TestService_ResolveTagNames(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockTagStore)
	svc := tag.NewService(mockStore)

	t.Run("resolves tag IDs to names", func(t *testing.T) {
		ids := []int64{1, 2}
		tags := []tag.Tag{
			{ID: 1, Name: "go"},
			{ID: 2, Name: "backend"},
		}
		mockStore.On("FindByIDs", ctx, ids).Return(tags, nil).Once()

		result, err := svc.ResolveTagNames(ctx, ids)

		assert.NoError(t, err)
		assert.Equal(t, []string{"go", "backend"}, result)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns empty slice for empty IDs", func(t *testing.T) {
		result, err := svc.ResolveTagNames(ctx, []int64{})

		assert.NoError(t, err)
		assert.Equal(t, []string{}, result)
	})
}
