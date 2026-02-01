// Package mocks provides mock implementations for testing
package mocks

import (
	"context"

	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockInsightCrawledContentStore is a mock implementation of insight.InsightCrawledContentStore
type MockInsightCrawledContentStore struct {
	mock.Mock
}

func (m *MockInsightCrawledContentStore) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]types.CrawledContent, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.CrawledContent), args.Error(1)
}

func (m *MockInsightCrawledContentStore) SearchSimilar(ctx context.Context, embedding []float32, limit int) ([]types.CrawledContent, error) {
	args := m.Called(ctx, embedding, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.CrawledContent), args.Error(1)
}

func (m *MockInsightCrawledContentStore) SearchSimilarByOrg(ctx context.Context, orgID uuid.UUID, embedding []float32, limit int) ([]types.CrawledContent, error) {
	args := m.Called(ctx, orgID, embedding, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.CrawledContent), args.Error(1)
}

func (m *MockInsightCrawledContentStore) FindRecentByOrg(ctx context.Context, orgID uuid.UUID, limit int) ([]types.CrawledContent, error) {
	args := m.Called(ctx, orgID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.CrawledContent), args.Error(1)
}
