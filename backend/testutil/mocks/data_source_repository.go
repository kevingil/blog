// Package mocks provides mock implementations for testing
package mocks

import (
	"context"
	"time"

	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockDataSourceRepository is a mock implementation of repository.DataSourceRepository
type MockDataSourceRepository struct {
	mock.Mock
}

func (m *MockDataSourceRepository) FindByID(ctx context.Context, id uuid.UUID) (*types.DataSource, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.DataSource), args.Error(1)
}

func (m *MockDataSourceRepository) FindByOrganizationID(ctx context.Context, orgID uuid.UUID) ([]types.DataSource, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.DataSource), args.Error(1)
}

func (m *MockDataSourceRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]types.DataSource, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.DataSource), args.Error(1)
}

func (m *MockDataSourceRepository) FindByURL(ctx context.Context, url string) (*types.DataSource, error) {
	args := m.Called(ctx, url)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.DataSource), args.Error(1)
}

func (m *MockDataSourceRepository) FindDueToCrawl(ctx context.Context, limit int) ([]types.DataSource, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.DataSource), args.Error(1)
}

func (m *MockDataSourceRepository) List(ctx context.Context, offset, limit int) ([]types.DataSource, int64, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]types.DataSource), args.Get(1).(int64), args.Error(2)
}

func (m *MockDataSourceRepository) Save(ctx context.Context, source *types.DataSource) error {
	args := m.Called(ctx, source)
	return args.Error(0)
}

func (m *MockDataSourceRepository) Update(ctx context.Context, source *types.DataSource) error {
	args := m.Called(ctx, source)
	return args.Error(0)
}

func (m *MockDataSourceRepository) UpdateCrawlStatus(ctx context.Context, id uuid.UUID, status string, errorMsg *string) error {
	args := m.Called(ctx, id, status, errorMsg)
	return args.Error(0)
}

func (m *MockDataSourceRepository) UpdateNextCrawlAt(ctx context.Context, id uuid.UUID, nextCrawlAt time.Time) error {
	args := m.Called(ctx, id, nextCrawlAt)
	return args.Error(0)
}

func (m *MockDataSourceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDataSourceRepository) IncrementContentCount(ctx context.Context, id uuid.UUID, delta int) error {
	args := m.Called(ctx, id, delta)
	return args.Error(0)
}
