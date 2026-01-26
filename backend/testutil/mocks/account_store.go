// Package mocks provides mock implementations for testing
package mocks

import (
	"context"

	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockAccountStore is a mock implementation of auth.AccountStore
type MockAccountStore struct {
	mock.Mock
}

func (m *MockAccountStore) FindByID(ctx context.Context, id uuid.UUID) (*types.Account, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Account), args.Error(1)
}

func (m *MockAccountStore) FindByEmail(ctx context.Context, email string) (*types.Account, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Account), args.Error(1)
}

func (m *MockAccountStore) Save(ctx context.Context, account *types.Account) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

func (m *MockAccountStore) Update(ctx context.Context, account *types.Account) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

func (m *MockAccountStore) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
