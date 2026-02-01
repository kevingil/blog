package organization_test

import (
	"context"
	"testing"
	"time"

	"backend/pkg/core"
	"backend/pkg/core/organization"
	"backend/pkg/types"
	"backend/testutil/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestService_GetByID(t *testing.T) {
	ctx := context.Background()

	t.Run("returns organization when found", func(t *testing.T) {
		mockOrgStore := new(mocks.MockOrganizationRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		svc := organization.NewService(mockOrgStore, mockAccountStore)

		orgID := uuid.New()
		bio := "Test bio"
		testOrg := &types.Organization{
			ID:        orgID,
			Name:      "Test Org",
			Slug:      "test-org",
			Bio:       &bio,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockOrgStore.On("FindByID", ctx, orgID).Return(testOrg, nil).Once()

		result, err := svc.GetByID(ctx, orgID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Test Org", result.Name)
		assert.Equal(t, "test-org", result.Slug)
		assert.Equal(t, "Test bio", result.Bio)
		mockOrgStore.AssertExpectations(t)
	})

	t.Run("returns error when organization not found", func(t *testing.T) {
		mockOrgStore := new(mocks.MockOrganizationRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		svc := organization.NewService(mockOrgStore, mockAccountStore)

		orgID := uuid.New()
		mockOrgStore.On("FindByID", ctx, orgID).Return(nil, core.ErrNotFound).Once()

		result, err := svc.GetByID(ctx, orgID)

		assert.ErrorIs(t, err, core.ErrNotFound)
		assert.Nil(t, result)
		mockOrgStore.AssertExpectations(t)
	})
}

func TestService_GetBySlug(t *testing.T) {
	ctx := context.Background()

	t.Run("returns organization when found", func(t *testing.T) {
		mockOrgStore := new(mocks.MockOrganizationRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		svc := organization.NewService(mockOrgStore, mockAccountStore)

		orgID := uuid.New()
		testOrg := &types.Organization{
			ID:        orgID,
			Name:      "Test Org",
			Slug:      "test-org",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockOrgStore.On("FindBySlug", ctx, "test-org").Return(testOrg, nil).Once()

		result, err := svc.GetBySlug(ctx, "test-org")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Test Org", result.Name)
		assert.Equal(t, "test-org", result.Slug)
		mockOrgStore.AssertExpectations(t)
	})

	t.Run("returns error when slug not found", func(t *testing.T) {
		mockOrgStore := new(mocks.MockOrganizationRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		svc := organization.NewService(mockOrgStore, mockAccountStore)

		mockOrgStore.On("FindBySlug", ctx, "nonexistent").Return(nil, core.ErrNotFound).Once()

		result, err := svc.GetBySlug(ctx, "nonexistent")

		assert.ErrorIs(t, err, core.ErrNotFound)
		assert.Nil(t, result)
		mockOrgStore.AssertExpectations(t)
	})
}

func TestService_List(t *testing.T) {
	ctx := context.Background()

	t.Run("returns list of organizations", func(t *testing.T) {
		mockOrgStore := new(mocks.MockOrganizationRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		svc := organization.NewService(mockOrgStore, mockAccountStore)

		orgs := []types.Organization{
			{
				ID:        uuid.New(),
				Name:      "Org One",
				Slug:      "org-one",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:        uuid.New(),
				Name:      "Org Two",
				Slug:      "org-two",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}

		mockOrgStore.On("List", ctx).Return(orgs, nil).Once()

		result, err := svc.List(ctx)

		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "Org One", result[0].Name)
		assert.Equal(t, "Org Two", result[1].Name)
		mockOrgStore.AssertExpectations(t)
	})
}

func TestService_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("creates organization successfully", func(t *testing.T) {
		mockOrgStore := new(mocks.MockOrganizationRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		svc := organization.NewService(mockOrgStore, mockAccountStore)

		bio := "A test organization"
		req := organization.CreateRequest{
			Name: "New Organization",
			Bio:  &bio,
		}

		// Slug check returns not found (slug is available)
		mockOrgStore.On("FindBySlug", ctx, "new-organization").Return(nil, core.ErrNotFound).Once()
		mockOrgStore.On("Save", ctx, mock.AnythingOfType("*types.Organization")).Return(nil).Once()

		result, err := svc.Create(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "New Organization", result.Name)
		assert.Equal(t, "new-organization", result.Slug)
		assert.Equal(t, "A test organization", result.Bio)
		mockOrgStore.AssertExpectations(t)
	})

	t.Run("creates organization with custom slug", func(t *testing.T) {
		mockOrgStore := new(mocks.MockOrganizationRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		svc := organization.NewService(mockOrgStore, mockAccountStore)

		req := organization.CreateRequest{
			Name: "New Organization",
			Slug: "custom-slug",
		}

		mockOrgStore.On("FindBySlug", ctx, "custom-slug").Return(nil, core.ErrNotFound).Once()
		mockOrgStore.On("Save", ctx, mock.AnythingOfType("*types.Organization")).Return(nil).Once()

		result, err := svc.Create(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "custom-slug", result.Slug)
		mockOrgStore.AssertExpectations(t)
	})

	t.Run("returns error when slug already exists", func(t *testing.T) {
		mockOrgStore := new(mocks.MockOrganizationRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		svc := organization.NewService(mockOrgStore, mockAccountStore)

		req := organization.CreateRequest{
			Name: "Existing Org",
			Slug: "existing-slug",
		}

		existingOrg := &types.Organization{
			ID:   uuid.New(),
			Name: "Existing",
			Slug: "existing-slug",
		}
		mockOrgStore.On("FindBySlug", ctx, "existing-slug").Return(existingOrg, nil).Once()

		result, err := svc.Create(ctx, req)

		assert.ErrorIs(t, err, core.ErrAlreadyExists)
		assert.Nil(t, result)
		mockOrgStore.AssertExpectations(t)
	})
}

func TestService_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("updates organization successfully", func(t *testing.T) {
		mockOrgStore := new(mocks.MockOrganizationRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		svc := organization.NewService(mockOrgStore, mockAccountStore)

		orgID := uuid.New()
		testOrg := &types.Organization{
			ID:        orgID,
			Name:      "Original Name",
			Slug:      "original-slug",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		newName := "Updated Name"
		req := organization.UpdateRequest{
			Name: &newName,
		}

		mockOrgStore.On("FindByID", ctx, orgID).Return(testOrg, nil).Once()
		mockOrgStore.On("Update", ctx, mock.AnythingOfType("*types.Organization")).Return(nil).Once()

		result, err := svc.Update(ctx, orgID, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Updated Name", result.Name)
		mockOrgStore.AssertExpectations(t)
	})

	t.Run("updates organization with new slug", func(t *testing.T) {
		mockOrgStore := new(mocks.MockOrganizationRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		svc := organization.NewService(mockOrgStore, mockAccountStore)

		orgID := uuid.New()
		testOrg := &types.Organization{
			ID:        orgID,
			Name:      "Original Name",
			Slug:      "original-slug",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		newSlug := "new-slug"
		req := organization.UpdateRequest{
			Slug: &newSlug,
		}

		mockOrgStore.On("FindByID", ctx, orgID).Return(testOrg, nil).Once()
		mockOrgStore.On("FindBySlug", ctx, "new-slug").Return(nil, core.ErrNotFound).Once()
		mockOrgStore.On("Update", ctx, mock.AnythingOfType("*types.Organization")).Return(nil).Once()

		result, err := svc.Update(ctx, orgID, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "new-slug", result.Slug)
		mockOrgStore.AssertExpectations(t)
	})

	t.Run("returns error when organization not found", func(t *testing.T) {
		mockOrgStore := new(mocks.MockOrganizationRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		svc := organization.NewService(mockOrgStore, mockAccountStore)

		orgID := uuid.New()
		newName := "Updated Name"
		req := organization.UpdateRequest{
			Name: &newName,
		}

		mockOrgStore.On("FindByID", ctx, orgID).Return(nil, core.ErrNotFound).Once()

		result, err := svc.Update(ctx, orgID, req)

		assert.ErrorIs(t, err, core.ErrNotFound)
		assert.Nil(t, result)
		mockOrgStore.AssertExpectations(t)
	})

	t.Run("returns error when new slug already exists", func(t *testing.T) {
		mockOrgStore := new(mocks.MockOrganizationRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		svc := organization.NewService(mockOrgStore, mockAccountStore)

		orgID := uuid.New()
		testOrg := &types.Organization{
			ID:        orgID,
			Name:      "Original Name",
			Slug:      "original-slug",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		existingOrg := &types.Organization{
			ID:   uuid.New(),
			Name: "Other Org",
			Slug: "taken-slug",
		}

		takenSlug := "taken-slug"
		req := organization.UpdateRequest{
			Slug: &takenSlug,
		}

		mockOrgStore.On("FindByID", ctx, orgID).Return(testOrg, nil).Once()
		mockOrgStore.On("FindBySlug", ctx, "taken-slug").Return(existingOrg, nil).Once()

		result, err := svc.Update(ctx, orgID, req)

		assert.ErrorIs(t, err, core.ErrAlreadyExists)
		assert.Nil(t, result)
		mockOrgStore.AssertExpectations(t)
	})
}

func TestService_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("deletes organization successfully", func(t *testing.T) {
		mockOrgStore := new(mocks.MockOrganizationRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		svc := organization.NewService(mockOrgStore, mockAccountStore)

		orgID := uuid.New()
		mockOrgStore.On("Delete", ctx, orgID).Return(nil).Once()

		err := svc.Delete(ctx, orgID)

		assert.NoError(t, err)
		mockOrgStore.AssertExpectations(t)
	})

	t.Run("returns error when organization not found", func(t *testing.T) {
		mockOrgStore := new(mocks.MockOrganizationRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		svc := organization.NewService(mockOrgStore, mockAccountStore)

		orgID := uuid.New()
		mockOrgStore.On("Delete", ctx, orgID).Return(core.ErrNotFound).Once()

		err := svc.Delete(ctx, orgID)

		assert.ErrorIs(t, err, core.ErrNotFound)
		mockOrgStore.AssertExpectations(t)
	})
}

func TestService_JoinOrganization(t *testing.T) {
	ctx := context.Background()

	t.Run("joins organization successfully", func(t *testing.T) {
		mockOrgStore := new(mocks.MockOrganizationRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		svc := organization.NewService(mockOrgStore, mockAccountStore)

		accountID := uuid.New()
		orgID := uuid.New()

		testOrg := &types.Organization{
			ID:   orgID,
			Name: "Test Org",
			Slug: "test-org",
		}

		testAccount := &types.Account{
			ID:             accountID,
			Name:           "Test User",
			OrganizationID: nil,
		}

		mockOrgStore.On("FindByID", ctx, orgID).Return(testOrg, nil).Once()
		mockAccountStore.On("FindByID", ctx, accountID).Return(testAccount, nil).Once()
		mockAccountStore.On("Update", ctx, mock.AnythingOfType("*types.Account")).Return(nil).Once()

		err := svc.JoinOrganization(ctx, accountID, orgID)

		assert.NoError(t, err)
		mockOrgStore.AssertExpectations(t)
		mockAccountStore.AssertExpectations(t)
	})

	t.Run("returns error when organization not found", func(t *testing.T) {
		mockOrgStore := new(mocks.MockOrganizationRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		svc := organization.NewService(mockOrgStore, mockAccountStore)

		accountID := uuid.New()
		orgID := uuid.New()

		mockOrgStore.On("FindByID", ctx, orgID).Return(nil, core.ErrNotFound).Once()

		err := svc.JoinOrganization(ctx, accountID, orgID)

		assert.ErrorIs(t, err, core.ErrNotFound)
		mockOrgStore.AssertExpectations(t)
	})

	t.Run("returns error when account not found", func(t *testing.T) {
		mockOrgStore := new(mocks.MockOrganizationRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		svc := organization.NewService(mockOrgStore, mockAccountStore)

		accountID := uuid.New()
		orgID := uuid.New()

		testOrg := &types.Organization{
			ID:   orgID,
			Name: "Test Org",
			Slug: "test-org",
		}

		mockOrgStore.On("FindByID", ctx, orgID).Return(testOrg, nil).Once()
		mockAccountStore.On("FindByID", ctx, accountID).Return(nil, core.ErrNotFound).Once()

		err := svc.JoinOrganization(ctx, accountID, orgID)

		assert.ErrorIs(t, err, core.ErrNotFound)
		mockOrgStore.AssertExpectations(t)
		mockAccountStore.AssertExpectations(t)
	})
}

func TestService_LeaveOrganization(t *testing.T) {
	ctx := context.Background()

	t.Run("leaves organization successfully", func(t *testing.T) {
		mockOrgStore := new(mocks.MockOrganizationRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		svc := organization.NewService(mockOrgStore, mockAccountStore)

		accountID := uuid.New()
		orgID := uuid.New()

		testAccount := &types.Account{
			ID:             accountID,
			Name:           "Test User",
			OrganizationID: &orgID,
		}

		mockAccountStore.On("FindByID", ctx, accountID).Return(testAccount, nil).Once()
		mockAccountStore.On("Update", ctx, mock.AnythingOfType("*types.Account")).Return(nil).Once()

		err := svc.LeaveOrganization(ctx, accountID)

		assert.NoError(t, err)
		mockAccountStore.AssertExpectations(t)
	})

	t.Run("returns error when account not found", func(t *testing.T) {
		mockOrgStore := new(mocks.MockOrganizationRepository)
		mockAccountStore := new(mocks.MockAccountRepository)
		svc := organization.NewService(mockOrgStore, mockAccountStore)

		accountID := uuid.New()

		mockAccountStore.On("FindByID", ctx, accountID).Return(nil, core.ErrNotFound).Once()

		err := svc.LeaveOrganization(ctx, accountID)

		assert.ErrorIs(t, err, core.ErrNotFound)
		mockAccountStore.AssertExpectations(t)
	})
}
