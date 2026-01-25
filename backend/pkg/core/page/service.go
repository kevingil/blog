package page

import (
	"context"
	"encoding/json"
	"math"

	"backend/pkg/core"
	"backend/pkg/database"
	"backend/pkg/database/models"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// CreateRequest represents a request to create a page
type CreateRequest struct {
	Slug        string                 `json:"slug" validate:"required,min=3,max=100"`
	Title       string                 `json:"title" validate:"required,min=3,max=200"`
	Content     string                 `json:"content" validate:"required,min=10"`
	Description string                 `json:"description" validate:"max=500"`
	ImageURL    string                 `json:"image_url" validate:"omitempty,url"`
	MetaData    map[string]interface{} `json:"meta_data"`
	IsPublished bool                   `json:"is_published"`
}

// UpdateRequest represents a request to update a page
type UpdateRequest struct {
	Title       *string                 `json:"title"`
	Content     *string                 `json:"content"`
	Description *string                 `json:"description"`
	ImageURL    *string                 `json:"image_url"`
	MetaData    *map[string]interface{} `json:"meta_data"`
	IsPublished *bool                   `json:"is_published"`
}

// ListResult represents the result of listing pages
type ListResult struct {
	Pages      []Page `json:"pages"`
	Total      int64  `json:"total"`
	Page       int    `json:"page"`
	PerPage    int    `json:"per_page"`
	TotalPages int    `json:"total_pages"`
}

// modelToPage converts a database model to domain type
func modelToPage(m *models.Page) *Page {
	if m == nil {
		return nil
	}

	var metaData map[string]interface{}
	if m.MetaData != nil {
		_ = json.Unmarshal(m.MetaData, &metaData)
	}

	return &Page{
		ID:          m.ID,
		Slug:        m.Slug,
		Title:       m.Title,
		Content:     m.Content,
		Description: m.Description,
		ImageURL:    m.ImageURL,
		MetaData:    metaData,
		IsPublished: m.IsPublished,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// GetByID retrieves a page by its ID
func GetByID(ctx context.Context, id uuid.UUID) (*Page, error) {
	db := database.DB()

	var model models.Page
	if err := db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return modelToPage(&model), nil
}

// GetBySlug retrieves a page by its slug
func GetBySlug(ctx context.Context, slug string) (*Page, error) {
	db := database.DB()

	var model models.Page
	if err := db.WithContext(ctx).Where("slug = ?", slug).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return modelToPage(&model), nil
}

// List retrieves pages with pagination and optional filters
func List(ctx context.Context, pageNum, perPage int, isPublished *bool) (*ListResult, error) {
	db := database.DB()

	// Apply defaults
	if perPage <= 0 {
		perPage = 20
	}
	if pageNum <= 0 {
		pageNum = 1
	}

	query := db.WithContext(ctx).Model(&models.Page{})

	// Apply filter if specified
	if isPublished != nil {
		query = query.Where("is_published = ?", *isPublished)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var pageModels []models.Page
	offset := (pageNum - 1) * perPage
	if err := query.Order("created_at DESC").Offset(offset).Limit(perPage).Find(&pageModels).Error; err != nil {
		return nil, err
	}

	pages := make([]Page, len(pageModels))
	for i, m := range pageModels {
		pages[i] = *modelToPage(&m)
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
func Create(ctx context.Context, req CreateRequest) (*Page, error) {
	db := database.DB()

	// Check if slug already exists
	var count int64
	db.WithContext(ctx).Model(&models.Page{}).Where("slug = ?", req.Slug).Count(&count)
	if count > 0 {
		return nil, core.ErrAlreadyExists
	}

	// Marshal metadata
	var metaDataJSON datatypes.JSON
	if req.MetaData != nil {
		data, err := json.Marshal(req.MetaData)
		if err != nil {
			return nil, err
		}
		metaDataJSON = datatypes.JSON(data)
	}

	model := models.Page{
		ID:          uuid.New(),
		Slug:        req.Slug,
		Title:       req.Title,
		Content:     req.Content,
		Description: req.Description,
		ImageURL:    req.ImageURL,
		MetaData:    metaDataJSON,
		IsPublished: req.IsPublished,
	}

	if err := db.WithContext(ctx).Create(&model).Error; err != nil {
		return nil, err
	}

	return modelToPage(&model), nil
}

// Update updates an existing page
func Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*Page, error) {
	db := database.DB()

	var model models.Page
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
	if req.Content != nil {
		model.Content = *req.Content
	}
	if req.Description != nil {
		model.Description = *req.Description
	}
	if req.ImageURL != nil {
		model.ImageURL = *req.ImageURL
	}
	if req.MetaData != nil {
		data, err := json.Marshal(*req.MetaData)
		if err != nil {
			return nil, err
		}
		model.MetaData = datatypes.JSON(data)
	}
	if req.IsPublished != nil {
		model.IsPublished = *req.IsPublished
	}

	if err := db.WithContext(ctx).Save(&model).Error; err != nil {
		return nil, err
	}

	return modelToPage(&model), nil
}

// Delete removes a page by its ID
func Delete(ctx context.Context, id uuid.UUID) error {
	db := database.DB()
	result := db.WithContext(ctx).Delete(&models.Page{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.ErrNotFound
	}
	return nil
}

// Legacy Service type for backward compatibility

type Service struct {
	store PageStore
}

func NewService(store PageStore) *Service {
	return &Service{store: store}
}
