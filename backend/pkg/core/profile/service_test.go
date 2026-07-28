package profile_test

import (
	"context"
	"testing"

	"backend/pkg/api/dto"
	"backend/pkg/core"
	"backend/pkg/core/profile"
	"backend/pkg/types"
	"backend/testutil/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestService_GetPublicProfile(t *testing.T) {
	ctx := context.Background()

	t.Run("returns public profile successfully", func(t *testing.T) {
		mockProfileStore := new(mocks.MockProfileRepository)
		mockSiteSettingsStore := new(mocks.MockSiteSettingsRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		mockOrgStore := new(mocks.MockOrganizationRepository)
		svc := profile.NewService(mockProfileStore, mockSiteSettingsStore, mockAccountStore, mockOrgStore)

		bio := "Test bio"
		imageURL := "https://example.com/image.jpg"
		emailPublic := "test@example.com"
		metaDesc := "Meta description"
		websiteURL := "https://example.com"
		publicProfile := &types.PublicProfile{
			Type:            "user",
			Name:            "Test User",
			Bio:             &bio,
			ImageURL:        &imageURL,
			EmailPublic:     &emailPublic,
			SocialLinks:     map[string]interface{}{"twitter": "testuser"},
			MetaDescription: &metaDesc,
			WebsiteURL:      &websiteURL,
		}

		mockProfileStore.On("GetPublicProfile", ctx).Return(publicProfile, nil).Once()

		result, err := svc.GetPublicProfile(ctx)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "user", result.Type)
		assert.Equal(t, "Test User", result.Name)
		assert.Equal(t, "Test bio", result.Bio)
		assert.Equal(t, "https://example.com/image.jpg", result.ImageURL)
		assert.Equal(t, "test@example.com", result.EmailPublic)
		assert.Equal(t, "testuser", result.SocialLinks["twitter"])
		assert.Equal(t, "Meta description", result.MetaDescription)
		mockProfileStore.AssertExpectations(t)
	})
}

func TestService_GetUserProfile(t *testing.T) {
	ctx := context.Background()

	t.Run("returns user profile successfully", func(t *testing.T) {
		mockProfileStore := new(mocks.MockProfileRepository)
		mockSiteSettingsStore := new(mocks.MockSiteSettingsRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		mockOrgStore := new(mocks.MockOrganizationRepository)
		svc := profile.NewService(mockProfileStore, mockSiteSettingsStore, mockAccountStore, mockOrgStore)

		accountID := uuid.New()
		orgID := uuid.New()
		bio := "User bio"
		profileImage := "https://example.com/profile.jpg"
		emailPublic := "user@example.com"
		metaDesc := "User meta description"
		testAccount := &types.Account{
			ID:              accountID,
			Name:            "Test User",
			Bio:             &bio,
			ProfileImage:    &profileImage,
			EmailPublic:     &emailPublic,
			SocialLinks:     map[string]interface{}{"github": "testuser"},
			MetaDescription: &metaDesc,
			OrganizationID:  &orgID,
		}

		mockAccountStore.On("FindByID", ctx, accountID).Return(testAccount, nil).Once()

		result, err := svc.GetUserProfile(ctx, accountID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, accountID, result.ID)
		assert.Equal(t, "Test User", result.Name)
		assert.Equal(t, "User bio", result.Bio)
		assert.Equal(t, "https://example.com/profile.jpg", result.ProfileImage)
		assert.Equal(t, "user@example.com", result.EmailPublic)
		assert.Equal(t, "testuser", result.SocialLinks["github"])
		assert.Equal(t, "User meta description", result.MetaDescription)
		assert.Equal(t, &orgID, result.OrganizationID)
		mockAccountStore.AssertExpectations(t)
	})

	t.Run("returns error when user not found", func(t *testing.T) {
		mockProfileStore := new(mocks.MockProfileRepository)
		mockSiteSettingsStore := new(mocks.MockSiteSettingsRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		mockOrgStore := new(mocks.MockOrganizationRepository)
		svc := profile.NewService(mockProfileStore, mockSiteSettingsStore, mockAccountStore, mockOrgStore)

		accountID := uuid.New()
		mockAccountStore.On("FindByID", ctx, accountID).Return(nil, core.ErrNotFound).Once()

		result, err := svc.GetUserProfile(ctx, accountID)

		assert.ErrorIs(t, err, core.ErrNotFound)
		assert.Nil(t, result)
		mockAccountStore.AssertExpectations(t)
	})
}

func TestService_UpdateUserProfile(t *testing.T) {
	ctx := context.Background()

	t.Run("updates user profile successfully", func(t *testing.T) {
		mockProfileStore := new(mocks.MockProfileRepository)
		mockSiteSettingsStore := new(mocks.MockSiteSettingsRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		mockOrgStore := new(mocks.MockOrganizationRepository)
		svc := profile.NewService(mockProfileStore, mockSiteSettingsStore, mockAccountStore, mockOrgStore)

		accountID := uuid.New()
		orgID := uuid.New()
		bio := "Original bio"
		profileImage := "https://example.com/original.jpg"
		emailPublic := "original@example.com"
		metaDesc := "Original meta"
		testAccount := &types.Account{
			ID:              accountID,
			Name:            "Original Name",
			Bio:             &bio,
			ProfileImage:    &profileImage,
			EmailPublic:     &emailPublic,
			SocialLinks:     map[string]interface{}{"twitter": "original"},
			MetaDescription: &metaDesc,
			OrganizationID:  &orgID,
		}

		newName := "Updated Name"
		newBio := "Updated bio"
		newProfileImage := "https://example.com/updated.jpg"
		newEmailPublic := "updated@example.com"
		newMetaDesc := "Updated meta"
		newSocialLinks := map[string]string{"twitter": "updated", "github": "newaccount"}
		updateReq := dto.ProfileUpdateRequest{
			Name:            &newName,
			Bio:             &newBio,
			ProfileImage:    &newProfileImage,
			EmailPublic:     &newEmailPublic,
			MetaDescription: &newMetaDesc,
			SocialLinks:     &newSocialLinks,
		}

		mockAccountStore.On("FindByID", ctx, accountID).Return(testAccount, nil).Once()
		mockAccountStore.On("Update", ctx, testAccount).Return(nil).Once()

		result, err := svc.UpdateUserProfile(ctx, accountID, updateReq)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Updated Name", result.Name)
		assert.Equal(t, "Updated bio", result.Bio)
		assert.Equal(t, "https://example.com/updated.jpg", result.ProfileImage)
		assert.Equal(t, "updated@example.com", result.EmailPublic)
		assert.Equal(t, "Updated meta", result.MetaDescription)
		assert.Equal(t, "updated", result.SocialLinks["twitter"])
		assert.Equal(t, "newaccount", result.SocialLinks["github"])
		mockAccountStore.AssertExpectations(t)
	})
}

func TestService_GetSiteSettings(t *testing.T) {
	ctx := context.Background()

	t.Run("returns site settings successfully", func(t *testing.T) {
		mockProfileStore := new(mocks.MockProfileRepository)
		mockSiteSettingsStore := new(mocks.MockSiteSettingsRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		mockOrgStore := new(mocks.MockOrganizationRepository)
		svc := profile.NewService(mockProfileStore, mockSiteSettingsStore, mockAccountStore, mockOrgStore)

		userID := uuid.New()
		orgID := uuid.New()
		testSettings := &types.SiteSettings{
			ID:                   1,
			PublicProfileType:    "organization",
			PublicUserID:         &userID,
			PublicOrganizationID: &orgID,
		}

		mockSiteSettingsStore.On("Get", ctx).Return(testSettings, nil).Once()

		result, err := svc.GetSiteSettings(ctx)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "organization", result.PublicProfileType)
		assert.Equal(t, &userID, result.PublicUserID)
		assert.Equal(t, &orgID, result.PublicOrganizationID)
		mockSiteSettingsStore.AssertExpectations(t)
	})

	t.Run("returns defaults when not found", func(t *testing.T) {
		mockProfileStore := new(mocks.MockProfileRepository)
		mockSiteSettingsStore := new(mocks.MockSiteSettingsRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		mockOrgStore := new(mocks.MockOrganizationRepository)
		svc := profile.NewService(mockProfileStore, mockSiteSettingsStore, mockAccountStore, mockOrgStore)

		mockSiteSettingsStore.On("Get", ctx).Return(nil, core.ErrNotFound).Once()

		result, err := svc.GetSiteSettings(ctx)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "user", result.PublicProfileType)
		assert.Nil(t, result.PublicUserID)
		assert.Nil(t, result.PublicOrganizationID)
		mockSiteSettingsStore.AssertExpectations(t)
	})
}

func TestService_UpdateSiteSettings(t *testing.T) {
	ctx := context.Background()

	t.Run("updates site settings successfully", func(t *testing.T) {
		mockProfileStore := new(mocks.MockProfileRepository)
		mockSiteSettingsStore := new(mocks.MockSiteSettingsRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		mockOrgStore := new(mocks.MockOrganizationRepository)
		svc := profile.NewService(mockProfileStore, mockSiteSettingsStore, mockAccountStore, mockOrgStore)

		userID := uuid.New()
		orgID := uuid.New()
		existingSettings := &types.SiteSettings{
			ID:                1,
			PublicProfileType: "user",
		}

		newProfileType := "organization"
		updateReq := dto.SiteSettingsUpdateRequest{
			PublicProfileType:    &newProfileType,
			PublicUserID:         &userID,
			PublicOrganizationID: &orgID,
		}

		testAccount := &types.Account{
			ID:   userID,
			Name: "Test User",
		}
		testOrg := &types.Organization{
			ID:   orgID,
			Name: "Test Org",
		}

		mockSiteSettingsStore.On("Get", ctx).Return(existingSettings, nil).Once()
		mockAccountStore.On("FindByID", ctx, userID).Return(testAccount, nil).Once()
		mockOrgStore.On("FindByID", ctx, orgID).Return(testOrg, nil).Once()
		mockSiteSettingsStore.On("Save", ctx, existingSettings).Return(nil).Once()

		result, err := svc.UpdateSiteSettings(ctx, updateReq)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "organization", result.PublicProfileType)
		assert.Equal(t, &userID, result.PublicUserID)
		assert.Equal(t, &orgID, result.PublicOrganizationID)
		mockSiteSettingsStore.AssertExpectations(t)
		mockAccountStore.AssertExpectations(t)
		mockOrgStore.AssertExpectations(t)
	})

	t.Run("creates settings when not found", func(t *testing.T) {
		mockProfileStore := new(mocks.MockProfileRepository)
		mockSiteSettingsStore := new(mocks.MockSiteSettingsRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		mockOrgStore := new(mocks.MockOrganizationRepository)
		svc := profile.NewService(mockProfileStore, mockSiteSettingsStore, mockAccountStore, mockOrgStore)

		newProfileType := "user"
		updateReq := dto.SiteSettingsUpdateRequest{
			PublicProfileType: &newProfileType,
		}

		mockSiteSettingsStore.On("Get", ctx).Return(nil, core.ErrNotFound).Once()
		mockSiteSettingsStore.On("Save", ctx, &types.SiteSettings{
			ID:                1,
			PublicProfileType: "user",
		}).Return(nil).Once()

		result, err := svc.UpdateSiteSettings(ctx, updateReq)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "user", result.PublicProfileType)
		mockSiteSettingsStore.AssertExpectations(t)
	})
}

func TestService_IsUserAdmin(t *testing.T) {
	ctx := context.Background()

	t.Run("returns true for admin user", func(t *testing.T) {
		mockProfileStore := new(mocks.MockProfileRepository)
		mockSiteSettingsStore := new(mocks.MockSiteSettingsRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		mockOrgStore := new(mocks.MockOrganizationRepository)
		svc := profile.NewService(mockProfileStore, mockSiteSettingsStore, mockAccountStore, mockOrgStore)

		userID := uuid.New()
		mockProfileStore.On("IsUserAdmin", ctx, userID).Return(true, nil).Once()

		isAdmin, err := svc.IsUserAdmin(ctx, userID)

		assert.NoError(t, err)
		assert.True(t, isAdmin)
		mockProfileStore.AssertExpectations(t)
	})

	t.Run("returns false for non-admin user", func(t *testing.T) {
		mockProfileStore := new(mocks.MockProfileRepository)
		mockSiteSettingsStore := new(mocks.MockSiteSettingsRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		mockOrgStore := new(mocks.MockOrganizationRepository)
		svc := profile.NewService(mockProfileStore, mockSiteSettingsStore, mockAccountStore, mockOrgStore)

		userID := uuid.New()
		mockProfileStore.On("IsUserAdmin", ctx, userID).Return(false, nil).Once()

		isAdmin, err := svc.IsUserAdmin(ctx, userID)

		assert.NoError(t, err)
		assert.False(t, isAdmin)
		mockProfileStore.AssertExpectations(t)
	})
}
