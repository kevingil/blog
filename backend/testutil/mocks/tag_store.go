// Package mocks provides mock implementations for testing
package mocks

import (
	"context"

	"backend/pkg/core/tag"

	"github.com/stretchr/testify/mock"
)

// MockTagStore is a mock implementation of tag.TagStore
type MockTagStore struct {
	mock.Mock
}

func (m *MockTagStore) FindByID(ctx context.Context, id int) (*tag.Tag, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tag.Tag), args.Error(1)
}

func (m *MockTagStore) FindByName(ctx context.Context, name string) (*tag.Tag, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tag.Tag), args.Error(1)
}

func (m *MockTagStore) FindByIDs(ctx context.Context, ids []int64) ([]tag.Tag, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).([]tag.Tag), args.Error(1)
}

func (m *MockTagStore) EnsureExists(ctx context.Context, names []string) ([]int64, error) {
	args := m.Called(ctx, names)
	return args.Get(0).([]int64), args.Error(1)
}

func (m *MockTagStore) List(ctx context.Context) ([]tag.Tag, error) {
	args := m.Called(ctx)
	return args.Get(0).([]tag.Tag), args.Error(1)
}

func (m *MockTagStore) Save(ctx context.Context, t *tag.Tag) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *MockTagStore) Delete(ctx context.Context, id int) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
