package source

import (
	"context"

	"github.com/google/uuid"
)

// Service provides business logic for article sources
type Service struct {
	store SourceStore
}

// NewService creates a new source service
func NewService(store SourceStore) *Service {
	return &Service{store: store}
}

// CreateRequest represents a request to create a source
type CreateRequest struct {
	ArticleID  uuid.UUID
	Title      string
	Content    string
	URL        string
	SourceType string
	Embedding  []float32
	MetaData   map[string]interface{}
}

// UpdateRequest represents a request to update a source
type UpdateRequest struct {
	Title      *string
	Content    *string
	URL        *string
	SourceType *string
	MetaData   *map[string]interface{}
}

// GetByID retrieves a source by its ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Source, error) {
	return s.store.FindByID(ctx, id)
}

// GetByArticleID retrieves all sources for an article
func (s *Service) GetByArticleID(ctx context.Context, articleID uuid.UUID) ([]Source, error) {
	return s.store.FindByArticleID(ctx, articleID)
}

// SearchSimilar performs vector similarity search for sources
func (s *Service) SearchSimilar(ctx context.Context, articleID uuid.UUID, embedding []float32, limit int) ([]Source, error) {
	if limit <= 0 {
		limit = 5
	}
	return s.store.SearchSimilar(ctx, articleID, embedding, limit)
}

// Create creates a new source
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Source, error) {
	sourceType := req.SourceType
	if sourceType == "" {
		sourceType = "web"
	}

	source := &Source{
		ID:         uuid.New(),
		ArticleID:  req.ArticleID,
		Title:      req.Title,
		Content:    req.Content,
		URL:        req.URL,
		SourceType: sourceType,
		Embedding:  req.Embedding,
		MetaData:   req.MetaData,
	}

	if err := s.store.Save(ctx, source); err != nil {
		return nil, err
	}

	return source, nil
}

// Update updates an existing source
func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*Source, error) {
	source, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.Title != nil {
		source.Title = *req.Title
	}
	if req.Content != nil {
		source.Content = *req.Content
	}
	if req.URL != nil {
		source.URL = *req.URL
	}
	if req.SourceType != nil {
		source.SourceType = *req.SourceType
	}
	if req.MetaData != nil {
		source.MetaData = *req.MetaData
	}

	if err := s.store.Update(ctx, source); err != nil {
		return nil, err
	}

	return source, nil
}

// Delete removes a source by its ID
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.store.Delete(ctx, id)
}

// UpdateEmbedding updates the embedding for a source
func (s *Service) UpdateEmbedding(ctx context.Context, id uuid.UUID, embedding []float32) error {
	source, err := s.store.FindByID(ctx, id)
	if err != nil {
		return err
	}

	source.Embedding = embedding
	return s.store.Update(ctx, source)
}
