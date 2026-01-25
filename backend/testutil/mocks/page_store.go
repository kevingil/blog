// Package mocks provides mock implementations for testing
package mocks

import (
	"context"

	"backend/pkg/core/page"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockPageStore is a mock implementation of page.PageStore
type MockPageStore struct {
	mock.Mock
}

func (m *MockPageStore) FindByID(ctx context.Context, id uuid.UUID) (*page.Page, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*page.Page), args.Error(1)
}

func (m *MockPageStore) FindBySlug(ctx context.Context, slug string) (*page.Page, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*page.Page), args.Error(1)
}

func (m *MockPageStore) List(ctx context.Context, opts page.ListOptions) ([]page.Page, int64, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).([]page.Page), args.Get(1).(int64), args.Error(2)
}

func (m *MockPageStore) Save(ctx context.Context, p *page.Page) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

func (m *MockPageStore) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
