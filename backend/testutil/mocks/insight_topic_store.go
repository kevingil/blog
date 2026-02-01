// Package mocks provides mock implementations for testing
package mocks

import (
	"context"
	"time"

	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockInsightTopicStore is a mock implementation of insight.InsightTopicStore
type MockInsightTopicStore struct {
	mock.Mock
}

func (m *MockInsightTopicStore) FindByID(ctx context.Context, id uuid.UUID) (*types.InsightTopic, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.InsightTopic), args.Error(1)
}

func (m *MockInsightTopicStore) FindByOrganizationID(ctx context.Context, orgID uuid.UUID) ([]types.InsightTopic, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.InsightTopic), args.Error(1)
}

func (m *MockInsightTopicStore) FindAll(ctx context.Context) ([]types.InsightTopic, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.InsightTopic), args.Error(1)
}

func (m *MockInsightTopicStore) SearchSimilar(ctx context.Context, embedding []float32, limit int, threshold float64) ([]types.InsightTopic, []float64, error) {
	args := m.Called(ctx, embedding, limit, threshold)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).([]types.InsightTopic), args.Get(1).([]float64), args.Error(2)
}

func (m *MockInsightTopicStore) Save(ctx context.Context, topic *types.InsightTopic) error {
	args := m.Called(ctx, topic)
	return args.Error(0)
}

func (m *MockInsightTopicStore) Update(ctx context.Context, topic *types.InsightTopic) error {
	args := m.Called(ctx, topic)
	return args.Error(0)
}

func (m *MockInsightTopicStore) UpdateContentCount(ctx context.Context, id uuid.UUID, count int) error {
	args := m.Called(ctx, id, count)
	return args.Error(0)
}

func (m *MockInsightTopicStore) UpdateLastInsightAt(ctx context.Context, id uuid.UUID, timestamp time.Time) error {
	args := m.Called(ctx, id, timestamp)
	return args.Error(0)
}

func (m *MockInsightTopicStore) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
