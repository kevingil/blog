package project

import (
	"context"
	"math"

	"backend/pkg/core"
	"backend/pkg/database/repository"
	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// CreateRequest represents a request to create a project
type CreateRequest struct {
	Title       string   `json:"title" validate:"required,min=3,max=200"`
	Description string   `json:"description" validate:"required,min=10,max=500"`
	Content     string   `json:"content"`
	Tags        []string `json:"tags" validate:"max=10,dive,min=2,max=30"`
	ImageURL    string   `json:"image_url" validate:"omitempty,url"`
	URL         string   `json:"url" validate:"omitempty,url"`
}

// UpdateRequest represents a request to update a project
type UpdateRequest struct {
	Title       *string   `json:"title"`
	Description *string   `json:"description"`
	Content     *string   `json:"content"`
	Tags        *[]string `json:"tags"`
	ImageURL    *string   `json:"image_url"`
	URL         *string   `json:"url"`
}

// ListResult represents the result of listing projects
type ListResult struct {
	Projects   []Project `json:"projects"`
	Total      int64     `json:"total"`
	Page       int       `json:"page"`
	PerPage    int       `json:"per_page"`
	TotalPages int       `json:"total_pages"`
}

// ProjectDetail includes project with resolved tag names
type ProjectDetail struct {
	Project Project  `json:"project"`
	Tags    []string `json:"tags"`
}

// Service provides business logic for projects
type Service struct {
	projectRepo repository.ProjectRepository
	tagRepo     repository.TagRepository
}

// NewService creates a new project service with the provided stores
func NewService(projectRepo repository.ProjectRepository, tagRepo repository.TagRepository) *Service {
	return &Service{
		projectRepo: projectRepo,
		tagRepo:     tagRepo,
	}
}

// GetByID retrieves a project by its ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Project, error) {
	return s.projectRepo.FindByID(ctx, id)
}

// GetDetail retrieves a project with resolved tag names
func (s *Service) GetDetail(ctx context.Context, id uuid.UUID) (*ProjectDetail, error) {
	project, err := s.projectRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	var tagNames []string
	if len(project.TagIDs) > 0 {
		tags, err := s.tagRepo.FindByIDs(ctx, project.TagIDs)
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

	opts := types.ProjectListOptions{
		Page:    pageNum,
		PerPage: perPage,
	}

	projects, total, err := s.projectRepo.List(ctx, opts)
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
	tagIDs, err := s.tagRepo.EnsureExists(ctx, req.Tags)
	if err != nil {
		return nil, err
	}

	project := &types.Project{
		ID:          uuid.New(),
		Title:       req.Title,
		Description: req.Description,
		Content:     req.Content,
		TagIDs:      pq.Int64Array(tagIDs),
		ImageURL:    req.ImageURL,
		URL:         req.URL,
	}

	if err := s.projectRepo.Save(ctx, project); err != nil {
		return nil, err
	}

	return project, nil
}

// Update updates an existing project
func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*Project, error) {
	project, err := s.projectRepo.FindByID(ctx, id)
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
		tagIDs, err := s.tagRepo.EnsureExists(ctx, *req.Tags)
		if err != nil {
			return nil, err
		}
		project.TagIDs = pq.Int64Array(tagIDs)
	}
	if req.ImageURL != nil {
		project.ImageURL = *req.ImageURL
	}
	if req.URL != nil {
		project.URL = *req.URL
	}

	if err := s.projectRepo.Update(ctx, project); err != nil {
		return nil, err
	}

	return project, nil
}

// Delete removes a project by its ID
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.projectRepo.Delete(ctx, id)
}
