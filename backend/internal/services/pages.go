package services

import (
	"blog-agent-go/backend/internal/database"
	"blog-agent-go/backend/internal/models"

	"gorm.io/gorm"
)

// PagesService provides methods to interact with the Page table
type PagesService struct {
	db database.Service
}

func NewPagesService(db database.Service) *PagesService {
	return &PagesService{db: db}
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

func (s *PagesService) GetAllPages() ([]models.Page, error) {
	db := s.db.GetDB()
	var pages []models.Page
	result := db.Find(&pages)
	if result.Error != nil {
		return nil, result.Error
	}
	return pages, nil
}
