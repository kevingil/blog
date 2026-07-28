// Package mocks provides mock implementations for testing
package mocks

import (
	"context"

	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockContentTopicMatchRepository is a mock implementation of repository.ContentTopicMatchRepository
type MockContentTopicMatchRepository struct {
	mock.Mock
}

func (m *MockContentTopicMatchRepository) SaveBatch(ctx context.Context, matches []types.ContentTopicMatch) error {
	args := m.Called(ctx, matches)
	return args.Error(0)
}

func (m *MockContentTopicMatchRepository) CountByTopicID(ctx context.Context, topicID uuid.UUID) (int64, error) {
	args := m.Called(ctx, topicID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockContentTopicMatchRepository) FindPrimaryByTopicID(ctx context.Context, topicID uuid.UUID, offset, limit int) ([]types.ContentTopicMatch, int64, error) {
	args := m.Called(ctx, topicID, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]types.ContentTopicMatch), args.Get(1).(int64), args.Error(2)
}
