package services

import (
	"backend/pkg/core"
	"backend/pkg/database"
	"backend/pkg/database/models"
	"fmt"
	"math"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// PagesService provides methods to interact with the Page table
type PagesService struct {
	db database.Service
}

func NewPagesService(db database.Service) *PagesService {
	return &PagesService{db: db}
}

type PageCreateRequest struct {
	Slug        string                 `json:"slug" validate:"required,min=3,max=100"`
	Title       string                 `json:"title" validate:"required,min=3,max=200"`
	Content     string                 `json:"content" validate:"required,min=10"`
	Description string                 `json:"description" validate:"max=500"`
	ImageURL    string                 `json:"image_url" validate:"omitempty,url"`
	MetaData    map[string]interface{} `json:"meta_data"`
	IsPublished bool                   `json:"is_published"`
}

type PageUpdateRequest struct {
	Title       *string                 `json:"title"`
	Content     *string                 `json:"content"`
	Description *string                 `json:"description"`
	ImageURL    *string                 `json:"image_url"`
	MetaData    *map[string]interface{} `json:"meta_data"`
	IsPublished *bool                   `json:"is_published"`
}

type PageListResponse struct {
	Pages      []models.Page `json:"pages"`
	Total      int64         `json:"total"`
	Page       int           `json:"page"`
	PerPage    int           `json:"per_page"`
	TotalPages int           `json:"total_pages"`
}

func (s *PagesService) GetPageBySlug(slug string) (*models.Page, error) {
	db := s.db.GetDB()
	var page models.Page
	result := db.Where("slug = ?", slug).First(&page)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &page, nil
}

func (s *PagesService) GetPageByID(id uuid.UUID) (*models.Page, error) {
	db := s.db.GetDB()
	var page models.Page
	result := db.Where("id = ?", id).First(&page)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &page, nil
}

func (s *PagesService) GetAllPages() ([]models.Page, error) {
	db := s.db.GetDB()
	var pages []models.Page
	result := db.Find(&pages)
	if result.Error != nil {
		return nil, result.Error
	}
	return pages, nil
}

func (s *PagesService) ListPagesWithPagination(page, perPage int, isPublished *bool) (*PageListResponse, error) {
	db := s.db.GetDB()

	if perPage <= 0 {
		perPage = 20
	}
	if page <= 0 {
		page = 1
	}

	query := db.Model(&models.Page{})

	// Filter by published status if specified
	if isPublished != nil {
		query = query.Where("is_published = ?", *isPublished)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var pages []models.Page
	offset := (page - 1) * perPage

	if err := query.Order("updated_at DESC").Offset(offset).Limit(perPage).Find(&pages).Error; err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(perPage)))

	response := &PageListResponse{
		Pages:      pages,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}

	return response, nil
}

func (s *PagesService) CreatePage(req PageCreateRequest) (*models.Page, error) {
	db := s.db.GetDB()

	// Check if slug already exists
	var existing models.Page
	if err := db.Where("slug = ?", req.Slug).First(&existing).Error; err == nil {
		return nil, core.AlreadyExistsError("Page with this slug")
	}

	var metaDataJSON datatypes.JSON
	if req.MetaData != nil {
		var err error
		metaDataJSON, err = datatypes.NewJSONType(req.MetaData).MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal meta_data: %w", err)
		}
	}

	page := models.Page{
		Slug:        req.Slug,
		Title:       req.Title,
		Content:     req.Content,
		Description: req.Description,
		ImageURL:    req.ImageURL,
		MetaData:    metaDataJSON,
		IsPublished: req.IsPublished,
	}

	if err := db.Create(&page).Error; err != nil {
		return nil, core.InternalError("Failed to create page")
	}

	return &page, nil
}

func (s *PagesService) UpdatePage(id uuid.UUID, req PageUpdateRequest) (*models.Page, error) {
	db := s.db.GetDB()

	var page models.Page
	if err := db.First(&page, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.NotFoundError("Page")
		}
		return nil, core.InternalError("Failed to fetch page")
	}

	updates := make(map[string]interface{})

	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Content != nil {
		updates["content"] = *req.Content
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.ImageURL != nil {
		updates["image_url"] = *req.ImageURL
	}
	if req.IsPublished != nil {
		updates["is_published"] = *req.IsPublished
	}
	if req.MetaData != nil {
		metaDataJSON, err := datatypes.NewJSONType(*req.MetaData).MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal meta_data: %w", err)
		}
		updates["meta_data"] = metaDataJSON
	}

	if len(updates) > 0 {
		if err := db.Model(&page).Updates(updates).Error; err != nil {
			return nil, core.InternalError("Failed to update page")
		}
	}

	// Reload the page to get updated values
	if err := db.First(&page, "id = ?", id).Error; err != nil {
		return nil, core.InternalError("Failed to reload page")
	}

	return &page, nil
}

func (s *PagesService) DeletePage(id uuid.UUID) error {
	db := s.db.GetDB()

	result := db.Delete(&models.Page{}, "id = ?", id)
	if result.Error != nil {
		return core.InternalError("Failed to delete page")
	}

	if result.RowsAffected == 0 {
		return core.NotFoundError("Page")
	}

	return nil
}
