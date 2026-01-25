package article

import (
	"context"
	"math"
	"regexp"
	"strings"

	"backend/pkg/core"
	"backend/pkg/core/tag"

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
	Publish  bool
	ImageURL string
}

// UpdateRequest represents a request to update an article
type UpdateRequest struct {
	Title    *string
	Content  *string
	Slug     *string
	Tags     *[]string
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

// VersionListResult represents the result of listing versions
type VersionListResult struct {
	Versions []ArticleVersion
	Total    int
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
func (s *Service) List(ctx context.Context, pageNum, perPage int, publishedOnly bool, authorID *uuid.UUID) (*ListResult, error) {
	if perPage <= 0 {
		perPage = 20
	}
	if pageNum <= 0 {
		pageNum = 1
	}

	opts := ListOptions{
		Page:          pageNum,
		PerPage:       perPage,
		PublishedOnly: publishedOnly,
		AuthorID:      authorID,
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
func (s *Service) Search(ctx context.Context, query string, pageNum, perPage int, publishedOnly bool) (*ListResult, error) {
	if perPage <= 0 {
		perPage = 20
	}
	if pageNum <= 0 {
		pageNum = 1
	}

	opts := SearchOptions{
		Query:         query,
		Page:          pageNum,
		PerPage:       perPage,
		PublishedOnly: publishedOnly,
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

// Create creates a new article as a draft
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
		ID:            uuid.New(),
		Slug:          slug,
		AuthorID:      req.AuthorID,
		TagIDs:        tagIDs,
		DraftTitle:    req.Title,
		DraftContent:  req.Content,
		DraftImageURL: req.ImageURL,
	}

	if err := s.store.Save(ctx, article); err != nil {
		return nil, err
	}

	// If publish is requested, publish immediately after creation
	if req.Publish {
		if err := s.store.Publish(ctx, article); err != nil {
			return nil, err
		}
	}

	return article, nil
}

// Update updates an existing article's draft content
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

	// Apply updates to draft fields
	if req.Title != nil {
		article.DraftTitle = *req.Title
	}
	if req.Content != nil {
		article.DraftContent = *req.Content
	}
	if req.Tags != nil {
		tagIDs, err := s.tagStore.EnsureExists(ctx, *req.Tags)
		if err != nil {
			return nil, err
		}
		article.TagIDs = tagIDs
	}
	if req.ImageURL != nil {
		article.DraftImageURL = *req.ImageURL
	}

	// Save draft (creates version asynchronously)
	if err := s.store.SaveDraft(ctx, article); err != nil {
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

// UpdateEmbedding updates the draft embedding for an article
func (s *Service) UpdateEmbedding(ctx context.Context, id uuid.UUID, embedding []float32) error {
	article, err := s.store.FindByID(ctx, id)
	if err != nil {
		return err
	}

	article.DraftEmbedding = embedding
	return s.store.Save(ctx, article)
}

// Publish publishes the current draft
func (s *Service) Publish(ctx context.Context, id uuid.UUID) (*Article, error) {
	article, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.store.Publish(ctx, article); err != nil {
		return nil, err
	}

	return article, nil
}

// Unpublish removes published status from an article
func (s *Service) Unpublish(ctx context.Context, id uuid.UUID) (*Article, error) {
	article, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if article.PublishedAt == nil {
		return nil, core.InvalidInputError("Article is not published")
	}

	if err := s.store.Unpublish(ctx, article); err != nil {
		return nil, err
	}

	return article, nil
}

// ListVersions retrieves all versions for an article
func (s *Service) ListVersions(ctx context.Context, id uuid.UUID) (*VersionListResult, error) {
	// Verify article exists
	_, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	versions, err := s.store.ListVersions(ctx, id)
	if err != nil {
		return nil, err
	}

	return &VersionListResult{
		Versions: versions,
		Total:    len(versions),
	}, nil
}

// GetVersion retrieves a specific version by ID
func (s *Service) GetVersion(ctx context.Context, versionID uuid.UUID) (*ArticleVersion, error) {
	return s.store.GetVersion(ctx, versionID)
}

// RevertToVersion creates a new draft by copying content from a historical version
func (s *Service) RevertToVersion(ctx context.Context, articleID, versionID uuid.UUID) (*Article, error) {
	// Verify article exists
	_, err := s.store.FindByID(ctx, articleID)
	if err != nil {
		return nil, err
	}

	if err := s.store.RevertToVersion(ctx, articleID, versionID); err != nil {
		return nil, err
	}

	// Return the updated article
	return s.store.FindByID(ctx, articleID)
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

// UpdateSessionMemory updates the session memory for an article
func (s *Service) UpdateSessionMemory(ctx context.Context, id uuid.UUID, sessionMemory map[string]interface{}) error {
	article, err := s.store.FindByID(ctx, id)
	if err != nil {
		return err
	}

	article.SessionMemory = sessionMemory
	return s.store.Save(ctx, article)
}
