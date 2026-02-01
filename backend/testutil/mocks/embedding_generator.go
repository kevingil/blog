// Package mocks provides mock implementations for testing
package mocks

import (
	"context"

	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/mock"
)

// MockEmbeddingGenerator is a mock implementation of insight.EmbeddingGenerator
type MockEmbeddingGenerator struct {
	mock.Mock
}

func (m *MockEmbeddingGenerator) GenerateEmbedding(ctx context.Context, text string) (pgvector.Vector, error) {
	args := m.Called(ctx, text)
	return args.Get(0).(pgvector.Vector), args.Error(1)
}
