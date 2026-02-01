// Package mocks provides mock implementations for testing
package mocks

import (
	"context"
	"time"

	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockDataSourceStore is a mock implementation of datasource.DataSourceStore
type MockDataSourceStore struct {
	mock.Mock
}

func (m *MockDataSourceStore) FindByID(ctx context.Context, id uuid.UUID) (*types.DataSource, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.DataSource), args.Error(1)
}

func (m *MockDataSourceStore) FindByOrganizationID(ctx context.Context, orgID uuid.UUID) ([]types.DataSource, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.DataSource), args.Error(1)
}

func (m *MockDataSourceStore) FindByUserID(ctx context.Context, userID uuid.UUID) ([]types.DataSource, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.DataSource), args.Error(1)
}

func (m *MockDataSourceStore) FindByURL(ctx context.Context, url string) (*types.DataSource, error) {
	args := m.Called(ctx, url)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.DataSource), args.Error(1)
}

func (m *MockDataSourceStore) FindDueToCrawl(ctx context.Context, limit int) ([]types.DataSource, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.DataSource), args.Error(1)
}

func (m *MockDataSourceStore) List(ctx context.Context, offset, limit int) ([]types.DataSource, int64, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]types.DataSource), args.Get(1).(int64), args.Error(2)
}

func (m *MockDataSourceStore) Save(ctx context.Context, source *types.DataSource) error {
	args := m.Called(ctx, source)
	return args.Error(0)
}

func (m *MockDataSourceStore) Update(ctx context.Context, source *types.DataSource) error {
	args := m.Called(ctx, source)
	return args.Error(0)
}

func (m *MockDataSourceStore) UpdateCrawlStatus(ctx context.Context, id uuid.UUID, status string, errorMsg *string) error {
	args := m.Called(ctx, id, status, errorMsg)
	return args.Error(0)
}

func (m *MockDataSourceStore) UpdateNextCrawlAt(ctx context.Context, id uuid.UUID, nextCrawlAt time.Time) error {
	args := m.Called(ctx, id, nextCrawlAt)
	return args.Error(0)
}

func (m *MockDataSourceStore) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
