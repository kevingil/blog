// Package mocks provides mock implementations for testing
package mocks

import (
	"context"

	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockInsightRepository is a mock implementation of repository.InsightRepository
type MockInsightRepository struct {
	mock.Mock
}

func (m *MockInsightRepository) FindByID(ctx context.Context, id uuid.UUID) (*types.Insight, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Insight), args.Error(1)
}

func (m *MockInsightRepository) List(ctx context.Context, offset, limit int) ([]types.Insight, int64, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]types.Insight), args.Get(1).(int64), args.Error(2)
}

func (m *MockInsightRepository) FindByOrganizationID(ctx context.Context, orgID uuid.UUID, offset, limit int) ([]types.Insight, int64, error) {
	args := m.Called(ctx, orgID, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]types.Insight), args.Get(1).(int64), args.Error(2)
}

func (m *MockInsightRepository) FindByTopicID(ctx context.Context, topicID uuid.UUID, offset, limit int) ([]types.Insight, int64, error) {
	args := m.Called(ctx, topicID, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]types.Insight), args.Get(1).(int64), args.Error(2)
}

func (m *MockInsightRepository) FindUnread(ctx context.Context, orgID uuid.UUID, limit int) ([]types.Insight, error) {
	args := m.Called(ctx, orgID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Insight), args.Error(1)
}

func (m *MockInsightRepository) SearchSimilar(ctx context.Context, embedding []float32, limit int) ([]types.Insight, error) {
	args := m.Called(ctx, embedding, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Insight), args.Error(1)
}

func (m *MockInsightRepository) SearchSimilarByOrg(ctx context.Context, orgID uuid.UUID, embedding []float32, limit int) ([]types.Insight, error) {
	args := m.Called(ctx, orgID, embedding, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Insight), args.Error(1)
}

func (m *MockInsightRepository) Save(ctx context.Context, insight *types.Insight) error {
	args := m.Called(ctx, insight)
	return args.Error(0)
}

func (m *MockInsightRepository) MarkAsRead(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockInsightRepository) TogglePinned(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockInsightRepository) MarkAsUsedInArticle(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockInsightRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockInsightRepository) CountUnread(ctx context.Context, orgID uuid.UUID) (int64, error) {
	args := m.Called(ctx, orgID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockInsightRepository) CountAllUnread(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}
