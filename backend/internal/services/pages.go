package services

import (
	"blog-agent-go/backend/internal/database"
	"blog-agent-go/backend/internal/models"
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
	Slug        string                 `json:"slug"`
	Title       string                 `json:"title"`
	Content     string                 `json:"content"`
	Description string                 `json:"description"`
	ImageURL    string                 `json:"image_url"`
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
	fmt.Println("\n=== ListPagesWithPagination ===")
	fmt.Printf("Input - page: %d, perPage: %d, isPublished: %v\n", page, perPage, isPublished)
	
	db := s.db.GetDB()

	if perPage <= 0 {
		perPage = 20
	}
	if page <= 0 {
		page = 1
	}

	fmt.Printf("After defaults - page: %d, perPage: %d\n", page, perPage)

	query := db.Model(&models.Page{})

	// Filter by published status if specified
	if isPublished != nil {
		fmt.Printf("Filtering by is_published = %v\n", *isPublished)
		query = query.Where("is_published = ?", *isPublished)
	} else {
		fmt.Println("No published filter applied")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		fmt.Printf("ERROR counting pages: %v\n", err)
		return nil, err
	}
	fmt.Printf("Total pages found: %d\n", total)

	var pages []models.Page
	offset := (page - 1) * perPage
	fmt.Printf("Query offset: %d, limit: %d\n", offset, perPage)
	
	if err := query.Order("updated_at DESC").Offset(offset).Limit(perPage).Find(&pages).Error; err != nil {
		fmt.Printf("ERROR fetching pages: %v\n", err)
		return nil, err
	}

	fmt.Printf("Pages fetched: %d\n", len(pages))
	for i, p := range pages {
		fmt.Printf("  [%d] ID: %s, Slug: %s, Title: %s, Published: %v\n", i, p.ID, p.Slug, p.Title, p.IsPublished)
	}

	totalPages := int(math.Ceil(float64(total) / float64(perPage)))
	fmt.Printf("Total pages calculation: %d\n", totalPages)

	response := &PageListResponse{
		Pages:      pages,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}
	
	fmt.Printf("Returning response with %d pages\n", len(response.Pages))
	fmt.Println("=== End ListPagesWithPagination ===\n")

	return response, nil
}

func (s *PagesService) CreatePage(req PageCreateRequest) (*models.Page, error) {
	fmt.Println("\n=== CreatePage ===")
	fmt.Printf("Request: Slug=%s, Title=%s, Published=%v\n", req.Slug, req.Title, req.IsPublished)
	
	db := s.db.GetDB()

	// Check if slug already exists
	var existing models.Page
	if err := db.Where("slug = ?", req.Slug).First(&existing).Error; err == nil {
		fmt.Printf("ERROR: Page with slug '%s' already exists\n", req.Slug)
		return nil, fmt.Errorf("page with slug '%s' already exists", req.Slug)
	}

	var metaDataJSON datatypes.JSON
	if req.MetaData != nil {
		fmt.Printf("Marshaling meta_data: %+v\n", req.MetaData)
		var err error
		metaDataJSON, err = datatypes.NewJSONType(req.MetaData).MarshalJSON()
		if err != nil {
			fmt.Printf("ERROR marshaling meta_data: %v\n", err)
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

	fmt.Printf("Creating page in database...\n")
	if err := db.Create(&page).Error; err != nil {
		fmt.Printf("ERROR creating page: %v\n", err)
		return nil, err
	}

	fmt.Printf("Page created successfully with ID: %s\n", page.ID)
	fmt.Println("=== End CreatePage ===\n")
	return &page, nil
}

func (s *PagesService) UpdatePage(id uuid.UUID, req PageUpdateRequest) (*models.Page, error) {
	fmt.Println("\n=== UpdatePage ===")
	fmt.Printf("Page ID: %s\n", id)
	
	db := s.db.GetDB()

	var page models.Page
	if err := db.First(&page, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			fmt.Printf("ERROR: Page not found with ID: %s\n", id)
			return nil, fmt.Errorf("page not found")
		}
		fmt.Printf("ERROR fetching page: %v\n", err)
		return nil, err
	}

	fmt.Printf("Found page: %s (slug: %s)\n", page.Title, page.Slug)

	updates := make(map[string]interface{})

	if req.Title != nil {
		fmt.Printf("Updating title: %s -> %s\n", page.Title, *req.Title)
		updates["title"] = *req.Title
	}
	if req.Content != nil {
		fmt.Printf("Updating content (length: %d -> %d)\n", len(page.Content), len(*req.Content))
		updates["content"] = *req.Content
	}
	if req.Description != nil {
		fmt.Printf("Updating description\n")
		updates["description"] = *req.Description
	}
	if req.ImageURL != nil {
		fmt.Printf("Updating image_url\n")
		updates["image_url"] = *req.ImageURL
	}
	if req.IsPublished != nil {
		fmt.Printf("Updating is_published: %v -> %v\n", page.IsPublished, *req.IsPublished)
		updates["is_published"] = *req.IsPublished
	}
	if req.MetaData != nil {
		fmt.Printf("Updating meta_data: %+v\n", *req.MetaData)
		metaDataJSON, err := datatypes.NewJSONType(*req.MetaData).MarshalJSON()
		if err != nil {
			fmt.Printf("ERROR marshaling meta_data: %v\n", err)
			return nil, fmt.Errorf("failed to marshal meta_data: %w", err)
		}
		updates["meta_data"] = metaDataJSON
	}

	fmt.Printf("Total updates to apply: %d\n", len(updates))

	if len(updates) > 0 {
		if err := db.Model(&page).Updates(updates).Error; err != nil {
			fmt.Printf("ERROR updating page: %v\n", err)
			return nil, err
		}
		fmt.Println("Updates applied successfully")
	} else {
		fmt.Println("No updates to apply")
	}

	// Reload the page to get updated values
	if err := db.First(&page, "id = ?", id).Error; err != nil {
		fmt.Printf("ERROR reloading page: %v\n", err)
		return nil, err
	}

	fmt.Printf("Page updated successfully\n")
	fmt.Println("=== End UpdatePage ===\n")
	return &page, nil
}

func (s *PagesService) DeletePage(id uuid.UUID) error {
	fmt.Println("\n=== DeletePage ===")
	fmt.Printf("Page ID: %s\n", id)
	
	db := s.db.GetDB()

	result := db.Delete(&models.Page{}, "id = ?", id)
	if result.Error != nil {
		fmt.Printf("ERROR deleting page: %v\n", result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		fmt.Printf("ERROR: No page found with ID: %s\n", id)
		return fmt.Errorf("page not found")
	}

	fmt.Printf("Page deleted successfully (rows affected: %d)\n", result.RowsAffected)
	fmt.Println("=== End DeletePage ===\n")
	return nil
}
