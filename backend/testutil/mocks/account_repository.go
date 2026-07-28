// Package mocks provides mock implementations for testing
package mocks

import (
	"context"

	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockAccountRepository is a mock implementation of repository.AccountRepository
type MockAccountRepository struct {
	mock.Mock
}

func (m *MockAccountRepository) FindByID(ctx context.Context, id uuid.UUID) (*types.Account, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Account), args.Error(1)
}

func (m *MockAccountRepository) FindByEmail(ctx context.Context, email string) (*types.Account, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Account), args.Error(1)
}

func (m *MockAccountRepository) Save(ctx context.Context, account *types.Account) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

func (m *MockAccountRepository) Update(ctx context.Context, account *types.Account) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

func (m *MockAccountRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
