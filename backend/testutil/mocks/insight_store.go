// Package mocks provides mock implementations for testing
package mocks

import (
	"context"

	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockInsightStore is a mock implementation of insight.InsightStore
type MockInsightStore struct {
	mock.Mock
}

func (m *MockInsightStore) FindByID(ctx context.Context, id uuid.UUID) (*types.Insight, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Insight), args.Error(1)
}

func (m *MockInsightStore) List(ctx context.Context, offset, limit int) ([]types.Insight, int64, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]types.Insight), args.Get(1).(int64), args.Error(2)
}

func (m *MockInsightStore) FindByOrganizationID(ctx context.Context, orgID uuid.UUID, offset, limit int) ([]types.Insight, int64, error) {
	args := m.Called(ctx, orgID, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]types.Insight), args.Get(1).(int64), args.Error(2)
}

func (m *MockInsightStore) FindByTopicID(ctx context.Context, topicID uuid.UUID, offset, limit int) ([]types.Insight, int64, error) {
	args := m.Called(ctx, topicID, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]types.Insight), args.Get(1).(int64), args.Error(2)
}

func (m *MockInsightStore) FindUnread(ctx context.Context, orgID uuid.UUID, limit int) ([]types.Insight, error) {
	args := m.Called(ctx, orgID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Insight), args.Error(1)
}

func (m *MockInsightStore) SearchSimilar(ctx context.Context, embedding []float32, limit int) ([]types.Insight, error) {
	args := m.Called(ctx, embedding, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Insight), args.Error(1)
}

func (m *MockInsightStore) SearchSimilarByOrg(ctx context.Context, orgID uuid.UUID, embedding []float32, limit int) ([]types.Insight, error) {
	args := m.Called(ctx, orgID, embedding, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Insight), args.Error(1)
}

func (m *MockInsightStore) Save(ctx context.Context, insight *types.Insight) error {
	args := m.Called(ctx, insight)
	return args.Error(0)
}

func (m *MockInsightStore) MarkAsRead(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockInsightStore) TogglePinned(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockInsightStore) MarkAsUsedInArticle(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockInsightStore) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockInsightStore) CountUnread(ctx context.Context, orgID uuid.UUID) (int64, error) {
	args := m.Called(ctx, orgID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockInsightStore) CountAllUnread(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}
