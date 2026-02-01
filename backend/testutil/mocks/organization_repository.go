// Package mocks provides mock implementations for testing
package mocks

import (
	"context"

	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockOrganizationRepository is a mock implementation of repository.OrganizationRepository
type MockOrganizationRepository struct {
	mock.Mock
}

func (m *MockOrganizationRepository) FindByID(ctx context.Context, id uuid.UUID) (*types.Organization, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Organization), args.Error(1)
}

func (m *MockOrganizationRepository) FindBySlug(ctx context.Context, slug string) (*types.Organization, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Organization), args.Error(1)
}

func (m *MockOrganizationRepository) List(ctx context.Context) ([]types.Organization, error) {
	args := m.Called(ctx)
	return args.Get(0).([]types.Organization), args.Error(1)
}

func (m *MockOrganizationRepository) Save(ctx context.Context, org *types.Organization) error {
	args := m.Called(ctx, org)
	return args.Error(0)
}

func (m *MockOrganizationRepository) Update(ctx context.Context, org *types.Organization) error {
	args := m.Called(ctx, org)
	return args.Error(0)
}

func (m *MockOrganizationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
