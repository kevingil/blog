package page

import (
	"context"
	"math"

	"backend/pkg/core"
	"backend/pkg/types"

	"github.com/google/uuid"
)

// Service provides business logic for pages
type Service struct {
	pageStore PageStore
}

// NewService creates a new page service with the provided store
func NewService(pageStore PageStore) *Service {
	return &Service{
		pageStore: pageStore,
	}
}

// GetByID retrieves a page by its ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*types.Page, error) {
	return s.pageStore.FindByID(ctx, id)
}

// GetBySlug retrieves a page by its slug
func (s *Service) GetBySlug(ctx context.Context, slug string) (*types.Page, error) {
	return s.pageStore.FindBySlug(ctx, slug)
}

// List retrieves pages with pagination and optional filters
func (s *Service) List(ctx context.Context, pageNum, perPage int, isPublished *bool) (*ListResult, error) {
	// Apply defaults
	if perPage <= 0 {
		perPage = 20
	}
	if pageNum <= 0 {
		pageNum = 1
	}

	opts := types.PageListOptions{
		Page:        pageNum,
		PerPage:     perPage,
		IsPublished: isPublished,
	}

	pages, total, err := s.pageStore.List(ctx, opts)
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
func (s *Service) Create(ctx context.Context, req CreateRequest) (*types.Page, error) {
	// Check if slug already exists
	existing, err := s.pageStore.FindBySlug(ctx, req.Slug)
	if err != nil && err != core.ErrNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, core.ErrAlreadyExists
	}

	page := &types.Page{
		ID:          uuid.New(),
		Slug:        req.Slug,
		Title:       req.Title,
		Content:     req.Content,
		Description: req.Description,
		ImageURL:    req.ImageURL,
		MetaData:    req.MetaData,
		IsPublished: req.IsPublished,
	}

	if err := s.pageStore.Save(ctx, page); err != nil {
		return nil, err
	}

	return page, nil
}

// Update updates an existing page
func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*types.Page, error) {
	page, err := s.pageStore.FindByID(ctx, id)
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

	if err := s.pageStore.Save(ctx, page); err != nil {
		return nil, err
	}

	return page, nil
}

// Delete removes a page by its ID
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.pageStore.Delete(ctx, id)
}
