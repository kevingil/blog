// Package mocks provides mock implementations for testing
package mocks

import (
	"context"

	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockSiteSettingsRepository is a mock implementation of repository.SiteSettingsRepository
type MockSiteSettingsRepository struct {
	mock.Mock
}

func (m *MockSiteSettingsRepository) Get(ctx context.Context) (*types.SiteSettings, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.SiteSettings), args.Error(1)
}

func (m *MockSiteSettingsRepository) Save(ctx context.Context, settings *types.SiteSettings) error {
	args := m.Called(ctx, settings)
	return args.Error(0)
}

// MockProfileRepository is a mock implementation of repository.ProfileRepository
type MockProfileRepository struct {
	mock.Mock
}

func (m *MockProfileRepository) GetPublicProfile(ctx context.Context) (*types.PublicProfile, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.PublicProfile), args.Error(1)
}

func (m *MockProfileRepository) IsUserAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, userID)
	return args.Bool(0), args.Error(1)
}
