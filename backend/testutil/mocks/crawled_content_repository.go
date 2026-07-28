// Package mocks provides mock implementations for testing
package mocks

import (
	"context"

	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockCrawledContentRepository is a mock implementation of repository.CrawledContentRepository
type MockCrawledContentRepository struct {
	mock.Mock
}

func (m *MockCrawledContentRepository) FindByDataSourceID(ctx context.Context, dsID uuid.UUID, offset, limit int) ([]types.CrawledContent, int64, error) {
	args := m.Called(ctx, dsID, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]types.CrawledContent), args.Get(1).(int64), args.Error(2)
}

func (m *MockCrawledContentRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]types.CrawledContent, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.CrawledContent), args.Error(1)
}

func (m *MockCrawledContentRepository) SearchSimilar(ctx context.Context, embedding []float32, limit int) ([]types.CrawledContent, error) {
	args := m.Called(ctx, embedding, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.CrawledContent), args.Error(1)
}

func (m *MockCrawledContentRepository) SearchSimilarByOrg(ctx context.Context, orgID uuid.UUID, embedding []float32, limit int) ([]types.CrawledContent, error) {
	args := m.Called(ctx, orgID, embedding, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.CrawledContent), args.Error(1)
}

func (m *MockCrawledContentRepository) FindRecentByOrg(ctx context.Context, orgID uuid.UUID, limit int) ([]types.CrawledContent, error) {
	args := m.Called(ctx, orgID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.CrawledContent), args.Error(1)
}

func (m *MockCrawledContentRepository) FindByID(ctx context.Context, id uuid.UUID) (*types.CrawledContent, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.CrawledContent), args.Error(1)
}

func (m *MockCrawledContentRepository) FindByURL(ctx context.Context, dataSourceID uuid.UUID, url string) (*types.CrawledContent, error) {
	args := m.Called(ctx, dataSourceID, url)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.CrawledContent), args.Error(1)
}

func (m *MockCrawledContentRepository) Save(ctx context.Context, content *types.CrawledContent) error {
	args := m.Called(ctx, content)
	return args.Error(0)
}

func (m *MockCrawledContentRepository) Update(ctx context.Context, content *types.CrawledContent) error {
	args := m.Called(ctx, content)
	return args.Error(0)
}

func (m *MockCrawledContentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCrawledContentRepository) DeleteByDataSourceID(ctx context.Context, dataSourceID uuid.UUID) error {
	args := m.Called(ctx, dataSourceID)
	return args.Error(0)
}

func (m *MockCrawledContentRepository) CountByDataSourceID(ctx context.Context, dataSourceID uuid.UUID) (int64, error) {
	args := m.Called(ctx, dataSourceID)
	return args.Get(0).(int64), args.Error(1)
}
