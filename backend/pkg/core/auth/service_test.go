package auth_test

import (
	"context"
	"testing"

	"backend/pkg/config"
	"backend/pkg/core"
	"backend/pkg/core/auth"
	"backend/pkg/types"
	"backend/testutil/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	config.SetTestDefaults()
}

// Helper function to create a hashed password for testing
func hashPassword(t *testing.T, password string) string {
	t.Helper()
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	return string(hashed)
}

func TestService_Login(t *testing.T) {
	ctx := context.Background()

	t.Run("returns token and user data on successful login", func(t *testing.T) {
		mockAccountStore := new(mocks.MockAccountStore)
		svc := auth.NewService(mockAccountStore)

		accountID := uuid.New()
		hashedPass := hashPassword(t, "correctpassword")
		testAccount := &types.Account{
			ID:           accountID,
			Name:         "Test User",
			Email:        "test@example.com",
			PasswordHash: hashedPass,
			Role:         "user",
		}

		mockAccountStore.On("FindByEmail", ctx, "test@example.com").Return(testAccount, nil).Once()

		result, err := svc.Login(ctx, types.LoginRequest{
			Email:    "test@example.com",
			Password: "correctpassword",
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Token)
		assert.Equal(t, accountID.String(), result.User.ID)
		assert.Equal(t, "Test User", result.User.Name)
		assert.Equal(t, "test@example.com", result.User.Email)
		assert.Equal(t, "user", result.User.Role)
		mockAccountStore.AssertExpectations(t)
	})

	t.Run("returns error when email not found", func(t *testing.T) {
		mockAccountStore := new(mocks.MockAccountStore)
		svc := auth.NewService(mockAccountStore)

		mockAccountStore.On("FindByEmail", ctx, "nonexistent@example.com").Return(nil, core.ErrNotFound).Once()

		result, err := svc.Login(ctx, types.LoginRequest{
			Email:    "nonexistent@example.com",
			Password: "anypassword",
		})

		assert.ErrorIs(t, err, core.ErrUnauthorized)
		assert.Nil(t, result)
		mockAccountStore.AssertExpectations(t)
	})

	t.Run("returns error when password is incorrect", func(t *testing.T) {
		mockAccountStore := new(mocks.MockAccountStore)
		svc := auth.NewService(mockAccountStore)

		accountID := uuid.New()
		hashedPass := hashPassword(t, "correctpassword")
		testAccount := &types.Account{
			ID:           accountID,
			Name:         "Test User",
			Email:        "test@example.com",
			PasswordHash: hashedPass,
			Role:         "user",
		}

		mockAccountStore.On("FindByEmail", ctx, "test@example.com").Return(testAccount, nil).Once()

		result, err := svc.Login(ctx, types.LoginRequest{
			Email:    "test@example.com",
			Password: "wrongpassword",
		})

		assert.ErrorIs(t, err, core.ErrUnauthorized)
		assert.Nil(t, result)
		mockAccountStore.AssertExpectations(t)
	})
}

func TestService_Register(t *testing.T) {
	ctx := context.Background()

	t.Run("creates new account successfully", func(t *testing.T) {
		mockAccountStore := new(mocks.MockAccountStore)
		svc := auth.NewService(mockAccountStore)

		mockAccountStore.On("FindByEmail", ctx, "newuser@example.com").Return(nil, core.ErrNotFound).Once()
		mockAccountStore.On("Save", ctx, mock.AnythingOfType("*types.Account")).Return(nil).Once()

		err := svc.Register(ctx, types.RegisterRequest{
			Name:     "New User",
			Email:    "newuser@example.com",
			Password: "securepassword",
		})

		assert.NoError(t, err)
		mockAccountStore.AssertExpectations(t)

		// Verify the account passed to Save had correct values
		savedAccount := mockAccountStore.Calls[1].Arguments.Get(1).(*types.Account)
		assert.Equal(t, "New User", savedAccount.Name)
		assert.Equal(t, "newuser@example.com", savedAccount.Email)
		assert.Equal(t, "user", savedAccount.Role)
		assert.NotEmpty(t, savedAccount.PasswordHash)
	})

	t.Run("returns error when email already exists", func(t *testing.T) {
		mockAccountStore := new(mocks.MockAccountStore)
		svc := auth.NewService(mockAccountStore)

		existingAccount := &types.Account{
			ID:    uuid.New(),
			Email: "existing@example.com",
		}

		mockAccountStore.On("FindByEmail", ctx, "existing@example.com").Return(existingAccount, nil).Once()

		err := svc.Register(ctx, types.RegisterRequest{
			Name:     "New User",
			Email:    "existing@example.com",
			Password: "password123",
		})

		assert.ErrorIs(t, err, core.ErrAlreadyExists)
		mockAccountStore.AssertExpectations(t)
	})
}

func TestService_GetAccount(t *testing.T) {
	ctx := context.Background()

	t.Run("returns account when found", func(t *testing.T) {
		mockAccountStore := new(mocks.MockAccountStore)
		svc := auth.NewService(mockAccountStore)

		accountID := uuid.New()
		testAccount := &types.Account{
			ID:    accountID,
			Name:  "Test User",
			Email: "test@example.com",
			Role:  "user",
		}

		mockAccountStore.On("FindByID", ctx, accountID).Return(testAccount, nil).Once()

		result, err := svc.GetAccount(ctx, accountID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, accountID, result.ID)
		assert.Equal(t, "Test User", result.Name)
		assert.Equal(t, "test@example.com", result.Email)
		mockAccountStore.AssertExpectations(t)
	})

	t.Run("returns error when account not found", func(t *testing.T) {
		mockAccountStore := new(mocks.MockAccountStore)
		svc := auth.NewService(mockAccountStore)

		accountID := uuid.New()
		mockAccountStore.On("FindByID", ctx, accountID).Return(nil, core.ErrNotFound).Once()

		result, err := svc.GetAccount(ctx, accountID)

		assert.ErrorIs(t, err, core.ErrNotFound)
		assert.Nil(t, result)
		mockAccountStore.AssertExpectations(t)
	})
}

func TestService_UpdateAccount(t *testing.T) {
	ctx := context.Background()

	t.Run("updates account successfully", func(t *testing.T) {
		mockAccountStore := new(mocks.MockAccountStore)
		svc := auth.NewService(mockAccountStore)

		accountID := uuid.New()
		existingAccount := &types.Account{
			ID:    accountID,
			Name:  "Old Name",
			Email: "old@example.com",
			Role:  "user",
		}

		mockAccountStore.On("FindByID", ctx, accountID).Return(existingAccount, nil).Once()
		mockAccountStore.On("Update", ctx, mock.AnythingOfType("*types.Account")).Return(nil).Once()

		err := svc.UpdateAccount(ctx, accountID, types.UpdateAccountRequest{
			Name:  "New Name",
			Email: "old@example.com", // Same email, no conflict check needed
		})

		assert.NoError(t, err)
		mockAccountStore.AssertExpectations(t)

		// Verify the account was updated with correct values
		updatedAccount := mockAccountStore.Calls[1].Arguments.Get(1).(*types.Account)
		assert.Equal(t, "New Name", updatedAccount.Name)
		assert.Equal(t, "old@example.com", updatedAccount.Email)
	})

	t.Run("updates account with new email when not taken", func(t *testing.T) {
		mockAccountStore := new(mocks.MockAccountStore)
		svc := auth.NewService(mockAccountStore)

		accountID := uuid.New()
		existingAccount := &types.Account{
			ID:    accountID,
			Name:  "Test User",
			Email: "old@example.com",
			Role:  "user",
		}

		mockAccountStore.On("FindByID", ctx, accountID).Return(existingAccount, nil).Once()
		mockAccountStore.On("FindByEmail", ctx, "new@example.com").Return(nil, core.ErrNotFound).Once()
		mockAccountStore.On("Update", ctx, mock.AnythingOfType("*types.Account")).Return(nil).Once()

		err := svc.UpdateAccount(ctx, accountID, types.UpdateAccountRequest{
			Name:  "Test User",
			Email: "new@example.com",
		})

		assert.NoError(t, err)
		mockAccountStore.AssertExpectations(t)

		updatedAccount := mockAccountStore.Calls[2].Arguments.Get(1).(*types.Account)
		assert.Equal(t, "new@example.com", updatedAccount.Email)
	})

	t.Run("returns error when new email is taken by another account", func(t *testing.T) {
		mockAccountStore := new(mocks.MockAccountStore)
		svc := auth.NewService(mockAccountStore)

		accountID := uuid.New()
		otherAccountID := uuid.New()
		existingAccount := &types.Account{
			ID:    accountID,
			Name:  "Test User",
			Email: "old@example.com",
			Role:  "user",
		}
		otherAccount := &types.Account{
			ID:    otherAccountID,
			Email: "taken@example.com",
		}

		mockAccountStore.On("FindByID", ctx, accountID).Return(existingAccount, nil).Once()
		mockAccountStore.On("FindByEmail", ctx, "taken@example.com").Return(otherAccount, nil).Once()

		err := svc.UpdateAccount(ctx, accountID, types.UpdateAccountRequest{
			Name:  "Test User",
			Email: "taken@example.com",
		})

		assert.ErrorIs(t, err, core.ErrAlreadyExists)
		mockAccountStore.AssertExpectations(t)
	})
}

func TestService_UpdatePassword(t *testing.T) {
	ctx := context.Background()

	t.Run("updates password successfully", func(t *testing.T) {
		mockAccountStore := new(mocks.MockAccountStore)
		svc := auth.NewService(mockAccountStore)

		accountID := uuid.New()
		oldHashedPass := hashPassword(t, "oldpassword")
		testAccount := &types.Account{
			ID:           accountID,
			Name:         "Test User",
			Email:        "test@example.com",
			PasswordHash: oldHashedPass,
			Role:         "user",
		}

		mockAccountStore.On("FindByID", ctx, accountID).Return(testAccount, nil).Once()
		mockAccountStore.On("Update", ctx, mock.AnythingOfType("*types.Account")).Return(nil).Once()

		err := svc.UpdatePassword(ctx, accountID, types.UpdatePasswordRequest{
			CurrentPassword: "oldpassword",
			NewPassword:     "newpassword",
		})

		assert.NoError(t, err)
		mockAccountStore.AssertExpectations(t)

		// Verify the password was actually changed
		updatedAccount := mockAccountStore.Calls[1].Arguments.Get(1).(*types.Account)
		assert.NotEqual(t, oldHashedPass, updatedAccount.PasswordHash)
		// Verify new password works
		err = bcrypt.CompareHashAndPassword([]byte(updatedAccount.PasswordHash), []byte("newpassword"))
		assert.NoError(t, err)
	})

	t.Run("returns error when current password is wrong", func(t *testing.T) {
		mockAccountStore := new(mocks.MockAccountStore)
		svc := auth.NewService(mockAccountStore)

		accountID := uuid.New()
		hashedPass := hashPassword(t, "correctpassword")
		testAccount := &types.Account{
			ID:           accountID,
			Name:         "Test User",
			Email:        "test@example.com",
			PasswordHash: hashedPass,
			Role:         "user",
		}

		mockAccountStore.On("FindByID", ctx, accountID).Return(testAccount, nil).Once()

		err := svc.UpdatePassword(ctx, accountID, types.UpdatePasswordRequest{
			CurrentPassword: "wrongpassword",
			NewPassword:     "newpassword",
		})

		assert.ErrorIs(t, err, core.ErrUnauthorized)
		mockAccountStore.AssertExpectations(t)
	})

	t.Run("returns error when account not found", func(t *testing.T) {
		mockAccountStore := new(mocks.MockAccountStore)
		svc := auth.NewService(mockAccountStore)

		accountID := uuid.New()
		mockAccountStore.On("FindByID", ctx, accountID).Return(nil, core.ErrNotFound).Once()

		err := svc.UpdatePassword(ctx, accountID, types.UpdatePasswordRequest{
			CurrentPassword: "oldpassword",
			NewPassword:     "newpassword",
		})

		assert.ErrorIs(t, err, core.ErrNotFound)
		mockAccountStore.AssertExpectations(t)
	})
}

func TestService_DeleteAccount(t *testing.T) {
	ctx := context.Background()

	t.Run("deletes account successfully", func(t *testing.T) {
		mockAccountStore := new(mocks.MockAccountStore)
		svc := auth.NewService(mockAccountStore)

		accountID := uuid.New()
		hashedPass := hashPassword(t, "correctpassword")
		testAccount := &types.Account{
			ID:           accountID,
			Name:         "Test User",
			Email:        "test@example.com",
			PasswordHash: hashedPass,
			Role:         "user",
		}

		mockAccountStore.On("FindByID", ctx, accountID).Return(testAccount, nil).Once()
		mockAccountStore.On("Delete", ctx, accountID).Return(nil).Once()

		err := svc.DeleteAccount(ctx, accountID, "correctpassword")

		assert.NoError(t, err)
		mockAccountStore.AssertExpectations(t)
	})

	t.Run("returns error when password is wrong", func(t *testing.T) {
		mockAccountStore := new(mocks.MockAccountStore)
		svc := auth.NewService(mockAccountStore)

		accountID := uuid.New()
		hashedPass := hashPassword(t, "correctpassword")
		testAccount := &types.Account{
			ID:           accountID,
			Name:         "Test User",
			Email:        "test@example.com",
			PasswordHash: hashedPass,
			Role:         "user",
		}

		mockAccountStore.On("FindByID", ctx, accountID).Return(testAccount, nil).Once()

		err := svc.DeleteAccount(ctx, accountID, "wrongpassword")

		assert.ErrorIs(t, err, core.ErrUnauthorized)
		mockAccountStore.AssertExpectations(t)
	})

	t.Run("returns error when account not found", func(t *testing.T) {
		mockAccountStore := new(mocks.MockAccountStore)
		svc := auth.NewService(mockAccountStore)

		accountID := uuid.New()
		mockAccountStore.On("FindByID", ctx, accountID).Return(nil, core.ErrNotFound).Once()

		err := svc.DeleteAccount(ctx, accountID, "anypassword")

		assert.ErrorIs(t, err, core.ErrNotFound)
		mockAccountStore.AssertExpectations(t)
	})
}
