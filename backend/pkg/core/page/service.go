package page

import (
	"context"
	"math"

	"backend/pkg/core"
	"backend/pkg/database"
	"backend/pkg/database/repository"
	"backend/pkg/types"

	"github.com/google/uuid"
)

// getRepo returns a page repository instance
func getRepo() *repository.PageRepository {
	return repository.NewPageRepository(database.DB())
}

// GetByID retrieves a page by its ID
func GetByID(ctx context.Context, id uuid.UUID) (*types.Page, error) {
	repo := getRepo()
	return repo.FindByID(ctx, id)
}

// GetBySlug retrieves a page by its slug
func GetBySlug(ctx context.Context, slug string) (*types.Page, error) {
	repo := getRepo()
	return repo.FindBySlug(ctx, slug)
}

// List retrieves pages with pagination and optional filters
func List(ctx context.Context, pageNum, perPage int, isPublished *bool) (*ListResult, error) {
	repo := getRepo()

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

	pages, total, err := repo.List(ctx, opts)
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
func Create(ctx context.Context, req CreateRequest) (*types.Page, error) {
	repo := getRepo()

	// Check if slug already exists
	existing, err := repo.FindBySlug(ctx, req.Slug)
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

	if err := repo.Save(ctx, page); err != nil {
		return nil, err
	}

	return page, nil
}

// Update updates an existing page
func Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*types.Page, error) {
	repo := getRepo()

	page, err := repo.FindByID(ctx, id)
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

	if err := repo.Save(ctx, page); err != nil {
		return nil, err
	}

	return page, nil
}

// Delete removes a page by its ID
func Delete(ctx context.Context, id uuid.UUID) error {
	repo := getRepo()
	return repo.Delete(ctx, id)
}

// Legacy Service type for backward compatibility

type Service struct {
	store PageStore
}

func NewService(store PageStore) *Service {
	return &Service{store: store}
}
