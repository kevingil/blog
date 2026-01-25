package auth_test

import (
	"context"
	"testing"

	"backend/pkg/core"
	"backend/pkg/core/auth"
	"backend/testutil/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

func TestService_Login(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockAccountStore)
	secretKey := "test-secret-key-for-testing"
	svc := auth.NewService(mockStore, secretKey)

	t.Run("returns token on successful login", func(t *testing.T) {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		account := &auth.Account{
			ID:           uuid.New(),
			Name:         "Test User",
			Email:        "test@example.com",
			PasswordHash: string(hashedPassword),
			Role:         "user",
		}
		mockStore.On("FindByEmail", ctx, "test@example.com").Return(account, nil).Once()

		result, err := svc.Login(ctx, auth.LoginRequest{
			Email:    "test@example.com",
			Password: "password123",
		})

		assert.NoError(t, err)
		assert.NotEmpty(t, result.Token)
		assert.Equal(t, account.Name, result.User.Name)
		assert.Equal(t, account.Email, result.User.Email)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error for invalid email", func(t *testing.T) {
		mockStore.On("FindByEmail", ctx, "wrong@example.com").Return(nil, core.ErrNotFound).Once()

		result, err := svc.Login(ctx, auth.LoginRequest{
			Email:    "wrong@example.com",
			Password: "password123",
		})

		assert.Nil(t, result)
		assert.ErrorIs(t, err, core.ErrUnauthorized)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error for invalid password", func(t *testing.T) {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
		account := &auth.Account{
			ID:           uuid.New(),
			Email:        "test@example.com",
			PasswordHash: string(hashedPassword),
		}
		mockStore.On("FindByEmail", ctx, "test@example.com").Return(account, nil).Once()

		result, err := svc.Login(ctx, auth.LoginRequest{
			Email:    "test@example.com",
			Password: "wrong-password",
		})

		assert.Nil(t, result)
		assert.ErrorIs(t, err, core.ErrUnauthorized)
		mockStore.AssertExpectations(t)
	})
}

func TestService_Register(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockAccountStore)
	secretKey := "test-secret-key"
	svc := auth.NewService(mockStore, secretKey)

	t.Run("registers user successfully", func(t *testing.T) {
		req := auth.RegisterRequest{
			Name:     "New User",
			Email:    "newuser@example.com",
			Password: "password123",
		}
		mockStore.On("FindByEmail", ctx, req.Email).Return(nil, core.ErrNotFound).Once()
		mockStore.On("Save", ctx, mock.AnythingOfType("*auth.Account")).Return(nil).Once()

		err := svc.Register(ctx, req)

		assert.NoError(t, err)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error when email already exists", func(t *testing.T) {
		existing := &auth.Account{ID: uuid.New(), Email: "existing@example.com"}
		mockStore.On("FindByEmail", ctx, "existing@example.com").Return(existing, nil).Once()

		err := svc.Register(ctx, auth.RegisterRequest{
			Name:     "User",
			Email:    "existing@example.com",
			Password: "password",
		})

		assert.ErrorIs(t, err, core.ErrAlreadyExists)
		mockStore.AssertExpectations(t)
	})
}

func TestService_ValidateToken(t *testing.T) {
	mockStore := new(mocks.MockAccountStore)
	secretKey := "test-secret-key"
	svc := auth.NewService(mockStore, secretKey)

	t.Run("validates valid token", func(t *testing.T) {
		ctx := context.Background()
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
		account := &auth.Account{
			ID:           uuid.New(),
			Email:        "test@example.com",
			PasswordHash: string(hashedPassword),
		}
		mockStore.On("FindByEmail", ctx, "test@example.com").Return(account, nil).Once()

		loginResp, _ := svc.Login(ctx, auth.LoginRequest{
			Email:    "test@example.com",
			Password: "password",
		})

		token, err := svc.ValidateToken(loginResp.Token)

		assert.NoError(t, err)
		assert.NotNil(t, token)
		assert.True(t, token.Valid)
	})

	t.Run("returns error for invalid token", func(t *testing.T) {
		token, err := svc.ValidateToken("invalid-token")

		assert.Nil(t, token)
		assert.ErrorIs(t, err, core.ErrUnauthorized)
	})
}

func TestService_UpdateAccount(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockAccountStore)
	secretKey := "test-secret-key"
	svc := auth.NewService(mockStore, secretKey)

	t.Run("updates account successfully", func(t *testing.T) {
		accountID := uuid.New()
		existing := &auth.Account{
			ID:    accountID,
			Name:  "Old Name",
			Email: "old@example.com",
		}
		req := auth.UpdateAccountRequest{
			Name:  "New Name",
			Email: "new@example.com",
		}
		mockStore.On("FindByID", ctx, accountID).Return(existing, nil).Once()
		mockStore.On("FindByEmail", ctx, "new@example.com").Return(nil, core.ErrNotFound).Once()
		mockStore.On("Update", ctx, existing).Return(nil).Once()

		err := svc.UpdateAccount(ctx, accountID, req)

		assert.NoError(t, err)
		assert.Equal(t, "New Name", existing.Name)
		assert.Equal(t, "new@example.com", existing.Email)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error when account not found", func(t *testing.T) {
		accountID := uuid.New()
		mockStore.On("FindByID", ctx, accountID).Return(nil, core.ErrNotFound).Once()

		err := svc.UpdateAccount(ctx, accountID, auth.UpdateAccountRequest{})

		assert.ErrorIs(t, err, core.ErrNotFound)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error when new email is taken", func(t *testing.T) {
		accountID := uuid.New()
		existing := &auth.Account{
			ID:    accountID,
			Email: "current@example.com",
		}
		otherAccount := &auth.Account{
			ID:    uuid.New(),
			Email: "taken@example.com",
		}
		mockStore.On("FindByID", ctx, accountID).Return(existing, nil).Once()
		mockStore.On("FindByEmail", ctx, "taken@example.com").Return(otherAccount, nil).Once()

		err := svc.UpdateAccount(ctx, accountID, auth.UpdateAccountRequest{
			Email: "taken@example.com",
		})

		assert.ErrorIs(t, err, core.ErrAlreadyExists)
		mockStore.AssertExpectations(t)
	})
}

func TestService_UpdatePassword(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockAccountStore)
	secretKey := "test-secret-key"
	svc := auth.NewService(mockStore, secretKey)

	t.Run("updates password successfully", func(t *testing.T) {
		accountID := uuid.New()
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("current-password"), bcrypt.DefaultCost)
		existing := &auth.Account{
			ID:           accountID,
			PasswordHash: string(hashedPassword),
		}
		mockStore.On("FindByID", ctx, accountID).Return(existing, nil).Once()
		mockStore.On("Update", ctx, existing).Return(nil).Once()

		err := svc.UpdatePassword(ctx, accountID, auth.UpdatePasswordRequest{
			CurrentPassword: "current-password",
			NewPassword:     "new-password",
		})

		assert.NoError(t, err)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error for wrong current password", func(t *testing.T) {
		accountID := uuid.New()
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.DefaultCost)
		existing := &auth.Account{
			ID:           accountID,
			PasswordHash: string(hashedPassword),
		}
		mockStore.On("FindByID", ctx, accountID).Return(existing, nil).Once()

		err := svc.UpdatePassword(ctx, accountID, auth.UpdatePasswordRequest{
			CurrentPassword: "wrong",
			NewPassword:     "new-password",
		})

		assert.ErrorIs(t, err, core.ErrUnauthorized)
		mockStore.AssertExpectations(t)
	})
}

func TestService_DeleteAccount(t *testing.T) {
	ctx := context.Background()
	mockStore := new(mocks.MockAccountStore)
	secretKey := "test-secret-key"
	svc := auth.NewService(mockStore, secretKey)

	t.Run("deletes account successfully", func(t *testing.T) {
		accountID := uuid.New()
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
		existing := &auth.Account{
			ID:           accountID,
			PasswordHash: string(hashedPassword),
		}
		mockStore.On("FindByID", ctx, accountID).Return(existing, nil).Once()
		mockStore.On("Delete", ctx, accountID).Return(nil).Once()

		err := svc.DeleteAccount(ctx, accountID, "password")

		assert.NoError(t, err)
		mockStore.AssertExpectations(t)
	})

	t.Run("returns error for wrong password", func(t *testing.T) {
		accountID := uuid.New()
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.DefaultCost)
		existing := &auth.Account{
			ID:           accountID,
			PasswordHash: string(hashedPassword),
		}
		mockStore.On("FindByID", ctx, accountID).Return(existing, nil).Once()

		err := svc.DeleteAccount(ctx, accountID, "wrong")

		assert.ErrorIs(t, err, core.ErrUnauthorized)
		mockStore.AssertExpectations(t)
	})
}
