// Package mocks provides mock implementations for testing
package mocks

import (
	"context"

	"backend/pkg/core/article"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockArticleStore is a mock implementation of article.ArticleStore
type MockArticleStore struct {
	mock.Mock
}

func (m *MockArticleStore) FindByID(ctx context.Context, id uuid.UUID) (*article.Article, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*article.Article), args.Error(1)
}

func (m *MockArticleStore) FindBySlug(ctx context.Context, slug string) (*article.Article, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*article.Article), args.Error(1)
}

func (m *MockArticleStore) List(ctx context.Context, opts article.ListOptions) ([]article.Article, int64, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).([]article.Article), args.Get(1).(int64), args.Error(2)
}

func (m *MockArticleStore) Search(ctx context.Context, opts article.SearchOptions) ([]article.Article, int64, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).([]article.Article), args.Get(1).(int64), args.Error(2)
}

func (m *MockArticleStore) SearchByEmbedding(ctx context.Context, embedding []float32, limit int) ([]article.Article, error) {
	args := m.Called(ctx, embedding, limit)
	return args.Get(0).([]article.Article), args.Error(1)
}

func (m *MockArticleStore) Save(ctx context.Context, a *article.Article) error {
	args := m.Called(ctx, a)
	return args.Error(0)
}

func (m *MockArticleStore) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockArticleStore) GetPopularTags(ctx context.Context, limit int) ([]int64, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]int64), args.Error(1)
}
