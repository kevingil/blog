// Package mocks provides mock implementations for testing
package mocks

import (
	"context"

	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockImageRepository is a mock implementation of repository.ImageRepository
type MockImageRepository struct {
	mock.Mock
}

func (m *MockImageRepository) FindByID(ctx context.Context, id uuid.UUID) (*types.ImageGeneration, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.ImageGeneration), args.Error(1)
}

func (m *MockImageRepository) FindByRequestID(ctx context.Context, requestID string) (*types.ImageGeneration, error) {
	args := m.Called(ctx, requestID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.ImageGeneration), args.Error(1)
}

func (m *MockImageRepository) Save(ctx context.Context, img *types.ImageGeneration) error {
	args := m.Called(ctx, img)
	return args.Error(0)
}

func (m *MockImageRepository) Update(ctx context.Context, img *types.ImageGeneration) error {
	args := m.Called(ctx, img)
	return args.Error(0)
}
