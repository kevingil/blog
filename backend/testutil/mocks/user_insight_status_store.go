// Package mocks provides mock implementations for testing
package mocks

import (
	"context"

	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockUserInsightStatusStore is a mock implementation of insight.UserInsightStatusStore
type MockUserInsightStatusStore struct {
	mock.Mock
}

func (m *MockUserInsightStatusStore) FindByUserAndInsight(ctx context.Context, userID, insightID uuid.UUID) (*types.UserInsightStatus, error) {
	args := m.Called(ctx, userID, insightID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.UserInsightStatus), args.Error(1)
}

func (m *MockUserInsightStatusStore) MarkAsRead(ctx context.Context, userID, insightID uuid.UUID) error {
	args := m.Called(ctx, userID, insightID)
	return args.Error(0)
}

func (m *MockUserInsightStatusStore) TogglePinned(ctx context.Context, userID, insightID uuid.UUID) (bool, error) {
	args := m.Called(ctx, userID, insightID)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockUserInsightStatusStore) MarkAsUsedInArticle(ctx context.Context, userID, insightID uuid.UUID) error {
	args := m.Called(ctx, userID, insightID)
	return args.Error(0)
}

func (m *MockUserInsightStatusStore) GetStatusMapForInsights(ctx context.Context, userID uuid.UUID, insightIDs []uuid.UUID) (map[uuid.UUID]*types.UserInsightStatus, error) {
	args := m.Called(ctx, userID, insightIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[uuid.UUID]*types.UserInsightStatus), args.Error(1)
}

func (m *MockUserInsightStatusStore) CountUnreadByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}
