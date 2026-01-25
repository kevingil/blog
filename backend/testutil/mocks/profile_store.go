// Package mocks provides mock implementations for testing
package mocks

import (
	"context"

	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockSiteSettingsStore is a mock implementation of profile.SiteSettingsStore
type MockSiteSettingsStore struct {
	mock.Mock
}

func (m *MockSiteSettingsStore) Get(ctx context.Context) (*types.SiteSettings, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.SiteSettings), args.Error(1)
}

func (m *MockSiteSettingsStore) Save(ctx context.Context, settings *types.SiteSettings) error {
	args := m.Called(ctx, settings)
	return args.Error(0)
}

// MockProfileStore is a mock implementation of profile.ProfileStore
type MockProfileStore struct {
	mock.Mock
}

func (m *MockProfileStore) GetPublicProfile(ctx context.Context) (*types.PublicProfile, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.PublicProfile), args.Error(1)
}

func (m *MockProfileStore) IsUserAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, userID)
	return args.Bool(0), args.Error(1)
}
