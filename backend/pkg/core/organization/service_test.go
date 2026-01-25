package organization_test

import (
	"context"
	"testing"

	"backend/pkg/core"
	"backend/pkg/core/auth"
	"backend/pkg/core/organization"
	"backend/testutil/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestService_GetByID(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockOrganizationStore)
	mockAccountStore := new(mocks.MockAccountStore)
	svc := organization.NewService(mockStore, mockAccountStore)

	t.Run("returns organization when found", func(t *testing.T) {
		orgID := uuid.New()
		expected := &organization.Organization{ID: orgID, Name: "Test Org"}
		mockStore.On("FindByID", ctx, orgID).Return(expected, nil).Once()

		result, err := svc.GetByID(ctx, orgID)

		assert.NoError(t, err)
		assert.Equal(t, expected, result)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		orgID := uuid.New()
		mockStore.On("FindByID", ctx, orgID).Return(nil, core.ErrNotFound).Once()

		result, err := svc.GetByID(ctx, orgID)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, core.ErrNotFound)
		mockStore.AssertExpectations(t)
	})
}

func TestService_GetBySlug(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockOrganizationStore)
	mockAccountStore := new(mocks.MockAccountStore)
	svc := organization.NewService(mockStore, mockAccountStore)

	t.Run("returns organization when found", func(t *testing.T) {
		slug := "test-org"
		expected := &organization.Organization{ID: uuid.New(), Slug: slug}
		mockStore.On("FindBySlug", ctx, slug).Return(expected, nil).Once()

		result, err := svc.GetBySlug(ctx, slug)

		assert.NoError(t, err)
		assert.Equal(t, expected, result)
		mockStore.AssertExpectations(t)
	})
}

func TestService_List(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockOrganizationStore)
	mockAccountStore := new(mocks.MockAccountStore)
	svc := organization.NewService(mockStore, mockAccountStore)

	t.Run("returns all organizations", func(t *testing.T) {
		orgs := []organization.Organization{
			{ID: uuid.New(), Name: "Org 1"},
			{ID: uuid.New(), Name: "Org 2"},
		}
		mockStore.On("List", ctx).Return(orgs, nil).Once()

		result, err := svc.List(ctx)

		assert.NoError(t, err)
		assert.Equal(t, orgs, result)
		mockStore.AssertExpectations(t)
	})
}

func TestService_Create(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockOrganizationStore)
	mockAccountStore := new(mocks.MockAccountStore)
	svc := organization.NewService(mockStore, mockAccountStore)

	t.Run("creates organization successfully", func(t *testing.T) {
		req := organization.CreateRequest{
			Name: "New Organization",
			Slug: "new-org",
		}
		mockStore.On("FindBySlug", ctx, "new-org").Return(nil, core.ErrNotFound).Once()
		mockStore.On("Save", ctx, mock.AnythingOfType("*organization.Organization")).Return(nil).Once()

		result, err := svc.Create(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, req.Name, result.Name)
		assert.Equal(t, req.Slug, result.Slug)
		mockStore.AssertExpectations(t)
	})

	t.Run("generates slug from name when not provided", func(t *testing.T) {
		req := organization.CreateRequest{
			Name: "My New Organization",
		}
		mockStore.On("FindBySlug", ctx, "my-new-organization").Return(nil, core.ErrNotFound).Once()
		mockStore.On("Save", ctx, mock.AnythingOfType("*organization.Organization")).Return(nil).Once()

		result, err := svc.Create(ctx, req)

		assert.NoError(t, err)
		assert.Equal(t, "my-new-organization", result.Slug)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error when slug already exists", func(t *testing.T) {
		req := organization.CreateRequest{
			Name: "Existing Org",
			Slug: "existing-org",
		}
		existing := &organization.Organization{ID: uuid.New(), Slug: req.Slug}
		mockStore.On("FindBySlug", ctx, req.Slug).Return(existing, nil).Once()

		result, err := svc.Create(ctx, req)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, core.ErrAlreadyExists)
		mockStore.AssertExpectations(t)
	})
}

func TestService_Update(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockOrganizationStore)
	mockAccountStore := new(mocks.MockAccountStore)
	svc := organization.NewService(mockStore, mockAccountStore)

	t.Run("updates organization successfully", func(t *testing.T) {
		orgID := uuid.New()
		existing := &organization.Organization{
			ID:   orgID,
			Name: "Old Name",
			Slug: "old-slug",
		}
		newName := "New Name"
		req := organization.UpdateRequest{
			Name: &newName,
		}
		mockStore.On("FindByID", ctx, orgID).Return(existing, nil).Once()
		mockStore.On("Update", ctx, existing).Return(nil).Once()

		result, err := svc.Update(ctx, orgID, req)

		assert.NoError(t, err)
		assert.Equal(t, newName, result.Name)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error when organization not found", func(t *testing.T) {
		orgID := uuid.New()
		req := organization.UpdateRequest{}
		mockStore.On("FindByID", ctx, orgID).Return(nil, core.ErrNotFound).Once()

		result, err := svc.Update(ctx, orgID, req)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, core.ErrNotFound)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error when new slug already exists", func(t *testing.T) {
		orgID := uuid.New()
		existing := &organization.Organization{
			ID:   orgID,
			Slug: "current-slug",
		}
		newSlug := "taken-slug"
		otherOrg := &organization.Organization{ID: uuid.New(), Slug: newSlug}
		req := organization.UpdateRequest{
			Slug: &newSlug,
		}
		mockStore.On("FindByID", ctx, orgID).Return(existing, nil).Once()
		mockStore.On("FindBySlug", ctx, newSlug).Return(otherOrg, nil).Once()

		result, err := svc.Update(ctx, orgID, req)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, core.ErrAlreadyExists)
		mockStore.AssertExpectations(t)
	})
}

func TestService_Delete(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockOrganizationStore)
	mockAccountStore := new(mocks.MockAccountStore)
	svc := organization.NewService(mockStore, mockAccountStore)

	t.Run("deletes organization successfully", func(t *testing.T) {
		orgID := uuid.New()
		mockStore.On("Delete", ctx, orgID).Return(nil).Once()

		err := svc.Delete(ctx, orgID)

		assert.NoError(t, err)
		mockStore.AssertExpectations(t)
	})
}

func TestService_JoinOrganization(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockOrganizationStore)
	mockAccountStore := new(mocks.MockAccountStore)
	svc := organization.NewService(mockStore, mockAccountStore)

	t.Run("joins organization successfully", func(t *testing.T) {
		accountID := uuid.New()
		orgID := uuid.New()
		org := &organization.Organization{ID: orgID}
		account := &auth.Account{ID: accountID}

		mockStore.On("FindByID", ctx, orgID).Return(org, nil).Once()
		mockAccountStore.On("FindByID", ctx, accountID).Return(account, nil).Once()
		mockAccountStore.On("Update", ctx, account).Return(nil).Once()

		err := svc.JoinOrganization(ctx, accountID, orgID)

		assert.NoError(t, err)
		assert.Equal(t, &orgID, account.OrganizationID)
		mockStore.AssertExpectations(t)
		mockAccountStore.AssertExpectations(t)
	})

	t.Run("returns error when organization not found", func(t *testing.T) {
		accountID := uuid.New()
		orgID := uuid.New()
		mockStore.On("FindByID", ctx, orgID).Return(nil, core.ErrNotFound).Once()

		err := svc.JoinOrganization(ctx, accountID, orgID)

		assert.ErrorIs(t, err, core.ErrNotFound)
		mockStore.AssertExpectations(t)
	})
}

func TestService_LeaveOrganization(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockOrganizationStore)
	mockAccountStore := new(mocks.MockAccountStore)
	svc := organization.NewService(mockStore, mockAccountStore)

	t.Run("leaves organization successfully", func(t *testing.T) {
		accountID := uuid.New()
		orgID := uuid.New()
		account := &auth.Account{ID: accountID, OrganizationID: &orgID}

		mockAccountStore.On("FindByID", ctx, accountID).Return(account, nil).Once()
		mockAccountStore.On("Update", ctx, account).Return(nil).Once()

		err := svc.LeaveOrganization(ctx, accountID)

		assert.NoError(t, err)
		assert.Nil(t, account.OrganizationID)
		mockAccountStore.AssertExpectations(t)
	})
}
