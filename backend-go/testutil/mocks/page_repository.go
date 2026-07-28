// Package mocks provides mock implementations for testing
package mocks

import (
	"context"

	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockPageRepository is a mock implementation of repository.PageRepository
type MockPageRepository struct {
	mock.Mock
}

func (m *MockPageRepository) FindByID(ctx context.Context, id uuid.UUID) (*types.Page, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Page), args.Error(1)
}

func (m *MockPageRepository) FindBySlug(ctx context.Context, slug string) (*types.Page, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Page), args.Error(1)
}

func (m *MockPageRepository) List(ctx context.Context, opts types.PageListOptions) ([]types.Page, int64, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).([]types.Page), args.Get(1).(int64), args.Error(2)
}

func (m *MockPageRepository) Save(ctx context.Context, p *types.Page) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

func (m *MockPageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
