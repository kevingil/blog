package image_test

import (
	"context"
	"testing"

	"backend/pkg/core"
	"backend/pkg/core/image"
	"backend/testutil/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestService_GetByID(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockImageStore)
	svc := image.NewService(mockStore)

	t.Run("returns image generation when found", func(t *testing.T) {
		imgID := uuid.New()
		expected := &image.ImageGeneration{ID: imgID, Prompt: "Test prompt"}
		mockStore.On("FindByID", ctx, imgID).Return(expected, nil).Once()

		result, err := svc.GetByID(ctx, imgID)

		assert.NoError(t, err)
		assert.Equal(t, expected, result)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		imgID := uuid.New()
		mockStore.On("FindByID", ctx, imgID).Return(nil, core.ErrNotFound).Once()

		result, err := svc.GetByID(ctx, imgID)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, core.ErrNotFound)
		mockStore.AssertExpectations(t)
	})
}

func TestService_GetByRequestID(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockImageStore)
	svc := image.NewService(mockStore)

	t.Run("returns image generation when found", func(t *testing.T) {
		requestID := "req-123"
		expected := &image.ImageGeneration{ID: uuid.New(), RequestID: requestID}
		mockStore.On("FindByRequestID", ctx, requestID).Return(expected, nil).Once()

		result, err := svc.GetByRequestID(ctx, requestID)

		assert.NoError(t, err)
		assert.Equal(t, expected, result)
		mockStore.AssertExpectations(t)
	})
}

func TestService_Create(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockImageStore)
	svc := image.NewService(mockStore)

	t.Run("creates image generation successfully", func(t *testing.T) {
		req := image.CreateRequest{
			Prompt:    "A beautiful sunset",
			Provider:  "openai",
			ModelName: "dall-e-3",
			RequestID: "req-456",
		}
		mockStore.On("Save", ctx, mock.AnythingOfType("*image.ImageGeneration")).Return(nil).Once()

		result, err := svc.Create(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, req.Prompt, result.Prompt)
		assert.Equal(t, req.Provider, result.Provider)
		assert.Equal(t, image.StatusPending, result.Status)
		mockStore.AssertExpectations(t)
	})
}

func TestService_MarkCompleted(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockImageStore)
	svc := image.NewService(mockStore)

	t.Run("marks image as completed", func(t *testing.T) {
		imgID := uuid.New()
		existing := &image.ImageGeneration{ID: imgID, Status: image.StatusPending}
		outputURL := "https://example.com/image.png"
		fileIndexID := uuid.New()

		mockStore.On("FindByID", ctx, imgID).Return(existing, nil).Once()
		mockStore.On("Update", ctx, existing).Return(nil).Once()

		err := svc.MarkCompleted(ctx, imgID, outputURL, &fileIndexID)

		assert.NoError(t, err)
		assert.Equal(t, image.StatusCompleted, existing.Status)
		assert.Equal(t, outputURL, existing.OutputURL)
		assert.NotNil(t, existing.CompletedAt)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error when image not found", func(t *testing.T) {
		imgID := uuid.New()
		mockStore.On("FindByID", ctx, imgID).Return(nil, core.ErrNotFound).Once()

		err := svc.MarkCompleted(ctx, imgID, "https://example.com/img.png", nil)

		assert.ErrorIs(t, err, core.ErrNotFound)
		mockStore.AssertExpectations(t)
	})
}

func TestService_MarkFailed(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockImageStore)
	svc := image.NewService(mockStore)

	t.Run("marks image as failed", func(t *testing.T) {
		imgID := uuid.New()
		existing := &image.ImageGeneration{ID: imgID, Status: image.StatusPending}
		errorMessage := "Generation failed"

		mockStore.On("FindByID", ctx, imgID).Return(existing, nil).Once()
		mockStore.On("Update", ctx, existing).Return(nil).Once()

		err := svc.MarkFailed(ctx, imgID, errorMessage)

		assert.NoError(t, err)
		assert.Equal(t, image.StatusFailed, existing.Status)
		assert.Equal(t, errorMessage, existing.ErrorMessage)
		assert.NotNil(t, existing.CompletedAt)
		mockStore.AssertExpectations(t)
	})
}

func TestService_GetStatus(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockImageStore)
	svc := image.NewService(mockStore)

	t.Run("returns status", func(t *testing.T) {
		imgID := uuid.New()
		existing := &image.ImageGeneration{ID: imgID, Status: image.StatusCompleted}
		mockStore.On("FindByID", ctx, imgID).Return(existing, nil).Once()

		status, err := svc.GetStatus(ctx, imgID)

		assert.NoError(t, err)
		assert.Equal(t, image.StatusCompleted, status)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error when image not found", func(t *testing.T) {
		imgID := uuid.New()
		mockStore.On("FindByID", ctx, imgID).Return(nil, core.ErrNotFound).Once()

		status, err := svc.GetStatus(ctx, imgID)

		assert.Empty(t, status)
		assert.ErrorIs(t, err, core.ErrNotFound)
		mockStore.AssertExpectations(t)
	})
}
