// Package mocks provides mock implementations for testing
package mocks

import (
	"context"

	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockContentTopicMatchStore is a mock implementation of insight.ContentTopicMatchStore
type MockContentTopicMatchStore struct {
	mock.Mock
}

func (m *MockContentTopicMatchStore) SaveBatch(ctx context.Context, matches []types.ContentTopicMatch) error {
	args := m.Called(ctx, matches)
	return args.Error(0)
}

func (m *MockContentTopicMatchStore) CountByTopicID(ctx context.Context, topicID uuid.UUID) (int64, error) {
	args := m.Called(ctx, topicID)
	return args.Get(0).(int64), args.Error(1)
}
