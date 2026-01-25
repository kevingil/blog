// Package mocks provides mock implementations for testing
package mocks

import (
	"context"

	"backend/pkg/core/organization"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockOrganizationStore is a mock implementation of organization.OrganizationStore
type MockOrganizationStore struct {
	mock.Mock
}

func (m *MockOrganizationStore) FindByID(ctx context.Context, id uuid.UUID) (*organization.Organization, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*organization.Organization), args.Error(1)
}

func (m *MockOrganizationStore) FindBySlug(ctx context.Context, slug string) (*organization.Organization, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*organization.Organization), args.Error(1)
}

func (m *MockOrganizationStore) List(ctx context.Context) ([]organization.Organization, error) {
	args := m.Called(ctx)
	return args.Get(0).([]organization.Organization), args.Error(1)
}

func (m *MockOrganizationStore) Save(ctx context.Context, org *organization.Organization) error {
	args := m.Called(ctx, org)
	return args.Error(0)
}

func (m *MockOrganizationStore) Update(ctx context.Context, org *organization.Organization) error {
	args := m.Called(ctx, org)
	return args.Error(0)
}

func (m *MockOrganizationStore) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
