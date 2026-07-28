// Package mocks provides mock implementations for testing
package mocks

import (
	"context"

	"backend/pkg/core/source"
	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockSourceRepository is a mock implementation of repository.SourceRepository
type MockSourceRepository struct {
	mock.Mock
}

func (m *MockSourceRepository) FindByID(ctx context.Context, id uuid.UUID) (*types.Source, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Source), args.Error(1)
}

func (m *MockSourceRepository) FindByArticleID(ctx context.Context, articleID uuid.UUID) ([]types.Source, error) {
	args := m.Called(ctx, articleID)
	return args.Get(0).([]types.Source), args.Error(1)
}

func (m *MockSourceRepository) SearchSimilar(ctx context.Context, articleID uuid.UUID, embedding []float32, limit int) ([]types.Source, error) {
	args := m.Called(ctx, articleID, embedding, limit)
	return args.Get(0).([]types.Source), args.Error(1)
}

func (m *MockSourceRepository) Save(ctx context.Context, s *types.Source) error {
	args := m.Called(ctx, s)
	return args.Error(0)
}

func (m *MockSourceRepository) Update(ctx context.Context, s *types.Source) error {
	args := m.Called(ctx, s)
	return args.Error(0)
}

func (m *MockSourceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSourceRepository) List(ctx context.Context, opts source.SourceListOptions) ([]source.SourceWithArticle, int64, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).([]source.SourceWithArticle), args.Get(1).(int64), args.Error(2)
}
