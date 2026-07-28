// Package mocks provides mock implementations for testing
package mocks

import (
	"context"

	"backend/pkg/types"

	"github.com/stretchr/testify/mock"
)

// MockTagRepository is a mock implementation of repository.TagRepository
type MockTagRepository struct {
	mock.Mock
}

func (m *MockTagRepository) FindByID(ctx context.Context, id int) (*types.Tag, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Tag), args.Error(1)
}

func (m *MockTagRepository) FindByName(ctx context.Context, name string) (*types.Tag, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Tag), args.Error(1)
}

func (m *MockTagRepository) FindByIDs(ctx context.Context, ids []int64) ([]types.Tag, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).([]types.Tag), args.Error(1)
}

func (m *MockTagRepository) EnsureExists(ctx context.Context, names []string) ([]int64, error) {
	args := m.Called(ctx, names)
	return args.Get(0).([]int64), args.Error(1)
}

func (m *MockTagRepository) List(ctx context.Context) ([]types.Tag, error) {
	args := m.Called(ctx)
	return args.Get(0).([]types.Tag), args.Error(1)
}

func (m *MockTagRepository) Save(ctx context.Context, t *types.Tag) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *MockTagRepository) Delete(ctx context.Context, id int) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
