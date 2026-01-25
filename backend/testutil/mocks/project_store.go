// Package mocks provides mock implementations for testing
package mocks

import (
	"context"

	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockProjectStore is a mock implementation of project.ProjectStore
type MockProjectStore struct {
	mock.Mock
}

func (m *MockProjectStore) FindByID(ctx context.Context, id uuid.UUID) (*types.Project, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Project), args.Error(1)
}

func (m *MockProjectStore) List(ctx context.Context, opts types.ProjectListOptions) ([]types.Project, int64, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).([]types.Project), args.Get(1).(int64), args.Error(2)
}

func (m *MockProjectStore) Save(ctx context.Context, p *types.Project) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

func (m *MockProjectStore) Update(ctx context.Context, p *types.Project) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

func (m *MockProjectStore) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
