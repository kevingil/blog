// Package ml provides machine learning utilities including embedding generation
package ml

import (
	"context"
	"fmt"

	openai "github.com/openai/openai-go"
	"github.com/pgvector/pgvector-go"
)

const (
	// EmbeddingModel is the OpenAI model used for generating embeddings
	EmbeddingModel = openai.EmbeddingModelTextEmbedding3Small
	// EmbeddingDimensions is the expected dimension count for embeddings
	EmbeddingDimensions = 1536
	// MaxEmbeddingTextLength is the maximum text length before truncation (~8192 tokens)
	MaxEmbeddingTextLength = 8000
)

// EmbeddingService provides embedding generation using OpenAI's API
type EmbeddingService struct {
	client *openai.Client
}

// NewEmbeddingService creates a new embedding service instance
func NewEmbeddingService() *EmbeddingService {
	client := openai.NewClient()
	return &EmbeddingService{
		client: &client,
	}
}

// GenerateEmbedding generates an embedding vector for the given text using OpenAI's API.
// The text is automatically truncated to MaxEmbeddingTextLength if too long.
// Returns a pgvector.Vector with EmbeddingDimensions (1536) dimensions.
func (s *EmbeddingService) GenerateEmbedding(ctx context.Context, text string) (pgvector.Vector, error) {
	if text == "" {
		return pgvector.Vector{}, fmt.Errorf("text cannot be empty")
	}

	// Truncate text if too long (OpenAI has token limits)
	// text-embedding-3-small supports up to 8192 tokens (~6000 characters)
	originalLength := len(text)
	if len(text) > MaxEmbeddingTextLength {
		text = text[:MaxEmbeddingTextLength]
	}

	// Generate embedding using OpenAI's text-embedding-3-small model
	// This model produces 1536-dimensional embeddings
	resp, err := s.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: []string{text},
		},
		Model: EmbeddingModel,
	})
	if err != nil {
		return pgvector.Vector{}, fmt.Errorf("failed to generate embedding from OpenAI (text length: %d): %w", originalLength, err)
	}

	if len(resp.Data) == 0 {
		return pgvector.Vector{}, fmt.Errorf("no embedding data returned from OpenAI")
	}

	// Validate embedding dimensions
	embeddingData := resp.Data[0].Embedding
	if len(embeddingData) != EmbeddingDimensions {
		return pgvector.Vector{}, fmt.Errorf("unexpected embedding dimensions: got %d, expected %d", len(embeddingData), EmbeddingDimensions)
	}

	// Convert []float64 to []float32 for pgvector compatibility
	embedding := make([]float32, len(embeddingData))
	for i, v := range embeddingData {
		embedding[i] = float32(v)
	}

	return pgvector.NewVector(embedding), nil
}

