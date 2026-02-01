// Package mocks provides mock implementations for testing
package mocks

import (
	"context"

	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockCrawledContentStore is a mock implementation of datasource.CrawledContentStore
type MockCrawledContentStore struct {
	mock.Mock
}

func (m *MockCrawledContentStore) FindByDataSourceID(ctx context.Context, dsID uuid.UUID, offset, limit int) ([]types.CrawledContent, int64, error) {
	args := m.Called(ctx, dsID, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]types.CrawledContent), args.Get(1).(int64), args.Error(2)
}
