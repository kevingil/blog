package profile_test

import (
	"context"
	"testing"

	"backend/pkg/core"
	"backend/pkg/core/profile"
	"backend/testutil/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestService_GetSiteSettings(t *testing.T) {
	ctx := context.Background()
	mockSettingsStore := new(mocks.MockSiteSettingsStore)
	mockProfileStore := new(mocks.MockProfileStore)
	svc := profile.NewService(mockSettingsStore, mockProfileStore)

	t.Run("returns site settings", func(t *testing.T) {
		expected := &profile.SiteSettings{ID: 1, PublicProfileType: "user"}
		mockSettingsStore.On("Get", ctx).Return(expected, nil).Once()

		result, err := svc.GetSiteSettings(ctx)

		assert.NoError(t, err)
		assert.Equal(t, expected, result)
		mockSettingsStore.AssertExpectations(t)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		mockSettingsStore.On("Get", ctx).Return(nil, core.ErrNotFound).Once()

		result, err := svc.GetSiteSettings(ctx)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, core.ErrNotFound)
		mockSettingsStore.AssertExpectations(t)
	})
}

func TestService_UpdateSiteSettings(t *testing.T) {
	ctx := context.Background()
	mockSettingsStore := new(mocks.MockSiteSettingsStore)
	mockProfileStore := new(mocks.MockProfileStore)
	svc := profile.NewService(mockSettingsStore, mockProfileStore)

	t.Run("updates site settings", func(t *testing.T) {
		settings := &profile.SiteSettings{ID: 1, PublicProfileType: "organization"}
		mockSettingsStore.On("Save", ctx, settings).Return(nil).Once()

		err := svc.UpdateSiteSettings(ctx, settings)

		assert.NoError(t, err)
		mockSettingsStore.AssertExpectations(t)
	})
}

func TestService_GetPublicProfile(t *testing.T) {
	ctx := context.Background()
	mockSettingsStore := new(mocks.MockSiteSettingsStore)
	mockProfileStore := new(mocks.MockProfileStore)
	svc := profile.NewService(mockSettingsStore, mockProfileStore)

	t.Run("returns public profile", func(t *testing.T) {
		expected := &profile.PublicProfile{Type: "user", Name: "Test User"}
		mockProfileStore.On("GetPublicProfile", ctx).Return(expected, nil).Once()

		result, err := svc.GetPublicProfile(ctx)

		assert.NoError(t, err)
		assert.Equal(t, expected, result)
		mockProfileStore.AssertExpectations(t)
	})
}

func TestService_IsUserAdmin(t *testing.T) {
	ctx := context.Background()
	mockSettingsStore := new(mocks.MockSiteSettingsStore)
	mockProfileStore := new(mocks.MockProfileStore)
	svc := profile.NewService(mockSettingsStore, mockProfileStore)

	t.Run("returns true for admin user", func(t *testing.T) {
		userID := uuid.New()
		mockProfileStore.On("IsUserAdmin", ctx, userID).Return(true, nil).Once()

		result, err := svc.IsUserAdmin(ctx, userID)

		assert.NoError(t, err)
		assert.True(t, result)
		mockProfileStore.AssertExpectations(t)
	})

	t.Run("returns false for non-admin user", func(t *testing.T) {
		userID := uuid.New()
		mockProfileStore.On("IsUserAdmin", ctx, userID).Return(false, nil).Once()

		result, err := svc.IsUserAdmin(ctx, userID)

		assert.NoError(t, err)
		assert.False(t, result)
		mockProfileStore.AssertExpectations(t)
	})
}

func TestService_UpdateSettings(t *testing.T) {
	ctx := context.Background()
	mockSettingsStore := new(mocks.MockSiteSettingsStore)
	mockProfileStore := new(mocks.MockProfileStore)
	svc := profile.NewService(mockSettingsStore, mockProfileStore)

	t.Run("updates existing settings", func(t *testing.T) {
		existing := &profile.SiteSettings{ID: 1, PublicProfileType: "user"}
		userID := uuid.New()
		req := profile.UpdateSiteSettingsRequest{
			PublicProfileType: "user",
			PublicUserID:      &userID,
		}
		mockSettingsStore.On("Get", ctx).Return(existing, nil).Once()
		mockSettingsStore.On("Save", ctx, existing).Return(nil).Once()

		err := svc.UpdateSettings(ctx, req)

		assert.NoError(t, err)
		assert.Equal(t, "user", existing.PublicProfileType)
		assert.Equal(t, &userID, existing.PublicUserID)
		mockSettingsStore.AssertExpectations(t)
	})

	t.Run("creates new settings when not found", func(t *testing.T) {
		req := profile.UpdateSiteSettingsRequest{
			PublicProfileType: "organization",
		}
		mockSettingsStore.On("Get", ctx).Return(nil, core.ErrNotFound).Once()
		mockSettingsStore.On("Save", ctx, &profile.SiteSettings{
			ID:                1,
			PublicProfileType: "organization",
		}).Return(nil).Once()

		err := svc.UpdateSettings(ctx, req)

		assert.NoError(t, err)
		mockSettingsStore.AssertExpectations(t)
	})
}
