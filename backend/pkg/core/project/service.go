package project

import (
	"context"
	"math"

	"backend/pkg/core"
	"backend/pkg/core/tag"

	"github.com/google/uuid"
)

// Service provides business logic for projects
type Service struct {
	store    ProjectStore
	tagStore tag.TagStore
}

// NewService creates a new project service
func NewService(store ProjectStore, tagStore tag.TagStore) *Service {
	return &Service{
		store:    store,
		tagStore: tagStore,
	}
}

// CreateRequest represents a request to create a project
type CreateRequest struct {
	Title       string
	Description string
	Content     string
	Tags        []string
	ImageURL    string
	URL         string
}

// UpdateRequest represents a request to update a project
type UpdateRequest struct {
	Title       *string
	Description *string
	Content     *string
	Tags        *[]string
	ImageURL    *string
	URL         *string
}

// ListResult represents the result of listing projects
type ListResult struct {
	Projects   []Project
	Total      int64
	Page       int
	PerPage    int
	TotalPages int
}

// ProjectDetail includes project with resolved tag names
type ProjectDetail struct {
	Project Project
	Tags    []string
}

// GetByID retrieves a project by its ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Project, error) {
	return s.store.FindByID(ctx, id)
}

// GetDetail retrieves a project with resolved tag names
func (s *Service) GetDetail(ctx context.Context, id uuid.UUID) (*ProjectDetail, error) {
	project, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	var tagNames []string
	if len(project.TagIDs) > 0 {
		tags, err := s.tagStore.FindByIDs(ctx, project.TagIDs)
		if err == nil {
			for _, t := range tags {
				tagNames = append(tagNames, t.Name)
			}
		}
	}

	return &ProjectDetail{
		Project: *project,
		Tags:    tagNames,
	}, nil
}

// List retrieves projects with pagination
func (s *Service) List(ctx context.Context, pageNum, perPage int) (*ListResult, error) {
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

	projects, total, err := s.store.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(perPage)))

	return &ListResult{
		Projects:   projects,
		Total:      total,
		Page:       pageNum,
		PerPage:    perPage,
		TotalPages: totalPages,
	}, nil
}

// Create creates a new project
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Project, error) {
	if req.Title == "" || req.Description == "" {
		return nil, core.ErrValidation
	}

	// Handle tags
	var tagIDs []int64
	if len(req.Tags) > 0 {
		var err error
		tagIDs, err = s.tagStore.EnsureExists(ctx, req.Tags)
		if err != nil {
			return nil, err
		}
	}

	project := &Project{
		ID:          uuid.New(),
		Title:       req.Title,
		Description: req.Description,
		Content:     req.Content,
		TagIDs:      tagIDs,
		ImageURL:    req.ImageURL,
		URL:         req.URL,
	}

	if err := s.store.Save(ctx, project); err != nil {
		return nil, err
	}

	return project, nil
}

// Update updates an existing project
func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*Project, error) {
	project, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.Title != nil {
		project.Title = *req.Title
	}
	if req.Description != nil {
		project.Description = *req.Description
	}
	if req.Content != nil {
		project.Content = *req.Content
	}
	if req.Tags != nil {
		tagIDs, err := s.tagStore.EnsureExists(ctx, *req.Tags)
		if err != nil {
			return nil, err
		}
		project.TagIDs = tagIDs
	}
	if req.ImageURL != nil {
		project.ImageURL = *req.ImageURL
	}
	if req.URL != nil {
		project.URL = *req.URL
	}

	if err := s.store.Update(ctx, project); err != nil {
		return nil, err
	}

	return project, nil
}

// Delete removes a project by its ID
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.store.Delete(ctx, id)
}
