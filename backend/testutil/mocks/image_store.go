// Package mocks provides mock implementations for testing
package mocks

import (
	"context"

	"backend/pkg/core/image"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockImageStore is a mock implementation of image.ImageStore
type MockImageStore struct {
	mock.Mock
}

func (m *MockImageStore) FindByID(ctx context.Context, id uuid.UUID) (*image.ImageGeneration, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*image.ImageGeneration), args.Error(1)
}

func (m *MockImageStore) FindByRequestID(ctx context.Context, requestID string) (*image.ImageGeneration, error) {
	args := m.Called(ctx, requestID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*image.ImageGeneration), args.Error(1)
}

func (m *MockImageStore) Save(ctx context.Context, img *image.ImageGeneration) error {
	args := m.Called(ctx, img)
	return args.Error(0)
}

func (m *MockImageStore) Update(ctx context.Context, img *image.ImageGeneration) error {
	args := m.Called(ctx, img)
	return args.Error(0)
}
