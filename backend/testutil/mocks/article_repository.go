// Package mocks provides mock implementations for testing
package mocks

import (
	"context"

	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockArticleRepository is a mock implementation of repository.ArticleRepository
type MockArticleRepository struct {
	mock.Mock
}

func (m *MockArticleRepository) FindByID(ctx context.Context, id uuid.UUID) (*types.Article, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Article), args.Error(1)
}

func (m *MockArticleRepository) FindBySlug(ctx context.Context, slug string) (*types.Article, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Article), args.Error(1)
}

func (m *MockArticleRepository) List(ctx context.Context, opts types.ArticleListOptions) ([]types.Article, int64, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).([]types.Article), args.Get(1).(int64), args.Error(2)
}

func (m *MockArticleRepository) Search(ctx context.Context, opts types.ArticleSearchOptions) ([]types.Article, int64, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).([]types.Article), args.Get(1).(int64), args.Error(2)
}

func (m *MockArticleRepository) SearchByEmbedding(ctx context.Context, embedding []float32, limit int) ([]types.Article, error) {
	args := m.Called(ctx, embedding, limit)
	return args.Get(0).([]types.Article), args.Error(1)
}

func (m *MockArticleRepository) Save(ctx context.Context, a *types.Article) error {
	args := m.Called(ctx, a)
	return args.Error(0)
}

func (m *MockArticleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockArticleRepository) GetPopularTags(ctx context.Context, limit int) ([]int64, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]int64), args.Error(1)
}

func (m *MockArticleRepository) SlugExists(ctx context.Context, slug string, excludeID *uuid.UUID) (bool, error) {
	args := m.Called(ctx, slug, excludeID)
	return args.Bool(0), args.Error(1)
}

// Version management methods

func (m *MockArticleRepository) SaveDraft(ctx context.Context, a *types.Article) error {
	args := m.Called(ctx, a)
	return args.Error(0)
}

func (m *MockArticleRepository) Publish(ctx context.Context, a *types.Article) error {
	args := m.Called(ctx, a)
	return args.Error(0)
}

func (m *MockArticleRepository) Unpublish(ctx context.Context, a *types.Article) error {
	args := m.Called(ctx, a)
	return args.Error(0)
}

func (m *MockArticleRepository) ListVersions(ctx context.Context, articleID uuid.UUID) ([]types.ArticleVersion, error) {
	args := m.Called(ctx, articleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.ArticleVersion), args.Error(1)
}

func (m *MockArticleRepository) GetVersion(ctx context.Context, versionID uuid.UUID) (*types.ArticleVersion, error) {
	args := m.Called(ctx, versionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.ArticleVersion), args.Error(1)
}

func (m *MockArticleRepository) RevertToVersion(ctx context.Context, articleID, versionID uuid.UUID) error {
	args := m.Called(ctx, articleID, versionID)
	return args.Error(0)
}
