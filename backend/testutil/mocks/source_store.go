// Package mocks provides mock implementations for testing
package mocks

import (
	"context"

	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockSourceStore is a mock implementation of source.SourceStore
type MockSourceStore struct {
	mock.Mock
}

func (m *MockSourceStore) FindByID(ctx context.Context, id uuid.UUID) (*types.Source, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Source), args.Error(1)
}

func (m *MockSourceStore) FindByArticleID(ctx context.Context, articleID uuid.UUID) ([]types.Source, error) {
	args := m.Called(ctx, articleID)
	return args.Get(0).([]types.Source), args.Error(1)
}

func (m *MockSourceStore) SearchSimilar(ctx context.Context, articleID uuid.UUID, embedding []float32, limit int) ([]types.Source, error) {
	args := m.Called(ctx, articleID, embedding, limit)
	return args.Get(0).([]types.Source), args.Error(1)
}

func (m *MockSourceStore) Save(ctx context.Context, s *types.Source) error {
	args := m.Called(ctx, s)
	return args.Error(0)
}

func (m *MockSourceStore) Update(ctx context.Context, s *types.Source) error {
	args := m.Called(ctx, s)
	return args.Error(0)
}

func (m *MockSourceStore) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
