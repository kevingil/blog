package project

import (
	"context"
	"math"
	"strings"

	"backend/pkg/core"
	"backend/pkg/database"
	"backend/pkg/database/models"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
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

// GetByID retrieves a project by its ID
func GetByID(ctx context.Context, id uuid.UUID) (*Project, error) {
	db := database.DB()
	var model models.Project
	if err := db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return modelToProject(&model), nil
}

// GetDetail retrieves a project with resolved tag names
func GetDetail(ctx context.Context, id uuid.UUID) (*ProjectDetail, error) {
	db := database.DB()

	var model models.Project
	if err := db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}

	var tagNames []string
	if len(model.TagIDs) > 0 {
		var tags []models.Tag
		if err := db.WithContext(ctx).Where("id IN ?", []int64(model.TagIDs)).Find(&tags).Error; err == nil {
			for _, t := range tags {
				tagNames = append(tagNames, t.Name)
			}
		}
	}

	return &ProjectDetail{
		Project: *modelToProject(&model),
		Tags:    tagNames,
	}, nil
}

// List retrieves projects with pagination
func List(ctx context.Context, pageNum, perPage int) (*ListResult, error) {
	db := database.DB()

	if perPage <= 0 {
		perPage = 20
	}
	if pageNum <= 0 {
		pageNum = 1
	}

	var total int64
	if err := db.WithContext(ctx).Model(&models.Project{}).Count(&total).Error; err != nil {
		return nil, err
	}

	var projectModels []models.Project
	offset := (pageNum - 1) * perPage
	if err := db.WithContext(ctx).Order("created_at DESC").Offset(offset).Limit(perPage).Find(&projectModels).Error; err != nil {
		return nil, err
	}

	projects := make([]Project, len(projectModels))
	for i, m := range projectModels {
		projects[i] = *modelToProject(&m)
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
func Create(ctx context.Context, req CreateRequest) (*Project, error) {
	db := database.DB()

	if req.Title == "" || req.Description == "" {
		return nil, core.ErrValidation
	}

	// Handle tags
	tagIDs, err := ensureTagsExist(ctx, db, req.Tags)
	if err != nil {
		return nil, err
	}

	model := &models.Project{
		ID:          uuid.New(),
		Title:       req.Title,
		Description: req.Description,
		Content:     req.Content,
		TagIDs:      pq.Int64Array(tagIDs),
		ImageURL:    req.ImageURL,
		URL:         req.URL,
	}

	if err := db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, err
	}

	return modelToProject(model), nil
}

// Update updates an existing project
func Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*Project, error) {
	db := database.DB()

	var model models.Project
	if err := db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}

	// Apply updates
	if req.Title != nil {
		model.Title = *req.Title
	}
	if req.Description != nil {
		model.Description = *req.Description
	}
	if req.Content != nil {
		model.Content = *req.Content
	}
	if req.Tags != nil {
		tagIDs, err := ensureTagsExist(ctx, db, *req.Tags)
		if err != nil {
			return nil, err
		}
		model.TagIDs = pq.Int64Array(tagIDs)
	}
	if req.ImageURL != nil {
		model.ImageURL = *req.ImageURL
	}
	if req.URL != nil {
		model.URL = *req.URL
	}

	if err := db.WithContext(ctx).Save(&model).Error; err != nil {
		return nil, err
	}

	return modelToProject(&model), nil
}

// Delete removes a project by its ID
func Delete(ctx context.Context, id uuid.UUID) error {
	db := database.DB()
	result := db.WithContext(ctx).Delete(&models.Project{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.ErrNotFound
	}
	return nil
}

// modelToProject converts a GORM model to a domain Project
func modelToProject(m *models.Project) *Project {
	return &Project{
		ID:          m.ID,
		Title:       m.Title,
		Description: m.Description,
		Content:     m.Content,
		TagIDs:      m.TagIDs,
		ImageURL:    m.ImageURL,
		URL:         m.URL,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// ensureTagsExist creates tags if they don't exist and returns their IDs
func ensureTagsExist(ctx context.Context, db *gorm.DB, names []string) ([]int64, error) {
	tagIDs := make([]int64, 0, len(names))

	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		var existingTag models.Tag
		err := db.WithContext(ctx).Where("LOWER(name) = LOWER(?)", name).First(&existingTag).Error
		if err == gorm.ErrRecordNotFound {
			newTag := &models.Tag{Name: name}
			if err := db.WithContext(ctx).Create(newTag).Error; err != nil {
				return nil, err
			}
			tagIDs = append(tagIDs, int64(newTag.ID))
		} else if err != nil {
			return nil, err
		} else {
			tagIDs = append(tagIDs, int64(existingTag.ID))
		}
	}

	return tagIDs, nil
}

