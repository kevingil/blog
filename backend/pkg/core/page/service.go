package page

import (
	"context"
	"math"

	"backend/pkg/core"

	"github.com/google/uuid"
)

// Service provides business logic for pages
type Service struct {
	store PageStore
}

// NewService creates a new page service
func NewService(store PageStore) *Service {
	return &Service{store: store}
}

// CreateRequest represents a request to create a page
type CreateRequest struct {
	Slug        string
	Title       string
	Content     string
	Description string
	ImageURL    string
	MetaData    map[string]interface{}
	IsPublished bool
}

// UpdateRequest represents a request to update a page
type UpdateRequest struct {
	Title       *string
	Content     *string
	Description *string
	ImageURL    *string
	MetaData    *map[string]interface{}
	IsPublished *bool
}

// ListResult represents the result of listing pages
type ListResult struct {
	Pages      []Page
	Total      int64
	Page       int
	PerPage    int
	TotalPages int
}

// GetByID retrieves a page by its ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Page, error) {
	return s.store.FindByID(ctx, id)
}

// GetBySlug retrieves a page by its slug
func (s *Service) GetBySlug(ctx context.Context, slug string) (*Page, error) {
	return s.store.FindBySlug(ctx, slug)
}

// List retrieves pages with pagination
func (s *Service) List(ctx context.Context, pageNum, perPage int) (*ListResult, error) {
	// Apply defaults
	if perPage <= 0 {
		perPage = 20
	}
	if pageNum <= 0 {
		pageNum = 1
	}

	opts := ListOptions{
		Page:    pageNum,
		PerPage: perPage,
	}

	pages, total, err := s.store.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(perPage)))

	return &ListResult{
		Pages:      pages,
		Total:      total,
		Page:       pageNum,
		PerPage:    perPage,
		TotalPages: totalPages,
	}, nil
}

// Create creates a new page
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Page, error) {
	// Check if slug already exists
	existing, err := s.store.FindBySlug(ctx, req.Slug)
	if err != nil && err != core.ErrNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, core.ErrAlreadyExists
	}

	page := &Page{
		ID:          uuid.New(),
		Slug:        req.Slug,
		Title:       req.Title,
		Content:     req.Content,
		Description: req.Description,
		ImageURL:    req.ImageURL,
		MetaData:    req.MetaData,
		IsPublished: req.IsPublished,
	}

	if err := s.store.Save(ctx, page); err != nil {
		return nil, err
	}

	return page, nil
}

// Update updates an existing page
func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*Page, error) {
	page, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.Title != nil {
		page.Title = *req.Title
	}
	if req.Content != nil {
		page.Content = *req.Content
	}
	if req.Description != nil {
		page.Description = *req.Description
	}
	if req.ImageURL != nil {
		page.ImageURL = *req.ImageURL
	}
	if req.MetaData != nil {
		page.MetaData = *req.MetaData
	}
	if req.IsPublished != nil {
		page.IsPublished = *req.IsPublished
	}

	if err := s.store.Save(ctx, page); err != nil {
		return nil, err
	}

	return page, nil
}

// Delete removes a page by its ID
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.store.Delete(ctx, id)
}
