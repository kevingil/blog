package article

import (
	"context"
	"math"
	"regexp"
	"strings"
	"time"

	"blog-agent-go/backend/internal/core"
	"blog-agent-go/backend/internal/core/tag"

	"github.com/google/uuid"
)

// Service provides business logic for articles
type Service struct {
	store    ArticleStore
	tagStore tag.TagStore
}

// NewService creates a new article service
func NewService(store ArticleStore, tagStore tag.TagStore) *Service {
	return &Service{
		store:    store,
		tagStore: tagStore,
	}
}

// CreateRequest represents a request to create an article
type CreateRequest struct {
	Title    string
	Content  string
	Slug     string
	AuthorID uuid.UUID
	Tags     []string
	IsDraft  bool
	ImageURL string
}

// UpdateRequest represents a request to update an article
type UpdateRequest struct {
	Title    *string
	Content  *string
	Slug     *string
	Tags     *[]string
	IsDraft  *bool
	ImageURL *string
}

// ListResult represents the result of listing articles
type ListResult struct {
	Articles   []Article
	Total      int64
	Page       int
	PerPage    int
	TotalPages int
}

// GetByID retrieves an article by its ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Article, error) {
	return s.store.FindByID(ctx, id)
}

// GetBySlug retrieves an article by its slug
func (s *Service) GetBySlug(ctx context.Context, slug string) (*Article, error) {
	return s.store.FindBySlug(ctx, slug)
}

// List retrieves articles with pagination and filtering
func (s *Service) List(ctx context.Context, pageNum, perPage int, isDraft *bool, authorID *uuid.UUID) (*ListResult, error) {
	if perPage <= 0 {
		perPage = 20
	}
	if pageNum <= 0 {
		pageNum = 1
	}

	opts := ListOptions{
		Page:     pageNum,
		PerPage:  perPage,
		IsDraft:  isDraft,
		AuthorID: authorID,
	}

	articles, total, err := s.store.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(perPage)))

	return &ListResult{
		Articles:   articles,
		Total:      total,
		Page:       pageNum,
		PerPage:    perPage,
		TotalPages: totalPages,
	}, nil
}

// Search performs full-text search on articles
func (s *Service) Search(ctx context.Context, query string, pageNum, perPage int, isDraft *bool) (*ListResult, error) {
	if perPage <= 0 {
		perPage = 20
	}
	if pageNum <= 0 {
		pageNum = 1
	}

	opts := SearchOptions{
		Query:   query,
		Page:    pageNum,
		PerPage: perPage,
		IsDraft: isDraft,
	}

	articles, total, err := s.store.Search(ctx, opts)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(perPage)))

	return &ListResult{
		Articles:   articles,
		Total:      total,
		Page:       pageNum,
		PerPage:    perPage,
		TotalPages: totalPages,
	}, nil
}

// SearchByEmbedding performs vector similarity search
func (s *Service) SearchByEmbedding(ctx context.Context, embedding []float32, limit int) ([]Article, error) {
	if limit <= 0 {
		limit = 5
	}
	return s.store.SearchByEmbedding(ctx, embedding, limit)
}

// Create creates a new article
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Article, error) {
	// Generate slug if not provided
	slug := req.Slug
	if slug == "" {
		slug = generateSlug(req.Title)
	}

	// Check if slug already exists
	existing, err := s.store.FindBySlug(ctx, slug)
	if err != nil && err != core.ErrNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, core.ErrAlreadyExists
	}

	// Handle tags
	var tagIDs []int64
	if len(req.Tags) > 0 {
		tagIDs, err = s.tagStore.EnsureExists(ctx, req.Tags)
		if err != nil {
			return nil, err
		}
	}

	article := &Article{
		ID:       uuid.New(),
		Slug:     slug,
		Title:    req.Title,
		Content:  req.Content,
		AuthorID: req.AuthorID,
		TagIDs:   tagIDs,
		IsDraft:  req.IsDraft,
		ImageURL: req.ImageURL,
	}

	// Set published date if not a draft
	if !req.IsDraft {
		now := time.Now()
		article.PublishedAt = &now
	}

	if err := s.store.Save(ctx, article); err != nil {
		return nil, err
	}

	return article, nil
}

// Update updates an existing article
func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*Article, error) {
	article, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if new slug is unique
	if req.Slug != nil && *req.Slug != article.Slug {
		existing, err := s.store.FindBySlug(ctx, *req.Slug)
		if err != nil && err != core.ErrNotFound {
			return nil, err
		}
		if existing != nil {
			return nil, core.ErrAlreadyExists
		}
		article.Slug = *req.Slug
	}

	// Apply updates
	if req.Title != nil {
		article.Title = *req.Title
	}
	if req.Content != nil {
		article.Content = *req.Content
	}
	if req.Tags != nil {
		tagIDs, err := s.tagStore.EnsureExists(ctx, *req.Tags)
		if err != nil {
			return nil, err
		}
		article.TagIDs = tagIDs
	}
	if req.ImageURL != nil {
		article.ImageURL = *req.ImageURL
	}
	if req.IsDraft != nil {
		wasDraft := article.IsDraft
		article.IsDraft = *req.IsDraft

		// Set published date when transitioning from draft to published
		if wasDraft && !*req.IsDraft && article.PublishedAt == nil {
			now := time.Now()
			article.PublishedAt = &now
		}
	}

	if err := s.store.Save(ctx, article); err != nil {
		return nil, err
	}

	return article, nil
}

// Delete removes an article by its ID
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.store.Delete(ctx, id)
}

// GetPopularTags returns popular tag IDs
func (s *Service) GetPopularTags(ctx context.Context, limit int) ([]int64, error) {
	if limit <= 0 {
		limit = 10
	}
	return s.store.GetPopularTags(ctx, limit)
}

// UpdateEmbedding updates the embedding for an article
func (s *Service) UpdateEmbedding(ctx context.Context, id uuid.UUID, embedding []float32) error {
	article, err := s.store.FindByID(ctx, id)
	if err != nil {
		return err
	}

	article.Embedding = embedding
	return s.store.Save(ctx, article)
}

// Helper function to generate slug
func generateSlug(title string) string {
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	reg := regexp.MustCompile("[^a-z0-9-]")
	slug = reg.ReplaceAllString(slug, "")
	reg = regexp.MustCompile("-+")
	slug = reg.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	return slug
}
