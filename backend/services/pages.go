package services

import (
	"blog-agent-go/backend/models"

	"gorm.io/gorm"
)

type PagesService struct {
	db *gorm.DB
}

func NewPagesService(db *gorm.DB) *PagesService {
	return &PagesService{db: db}
}

func (s *PagesService) GetAboutPage() (*models.AboutPage, error) {
	var page models.AboutPage
	result := s.db.First(&page)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &page, nil
}

func (s *PagesService) GetContactPage() (*models.ContactPage, error) {
	var page models.ContactPage
	result := s.db.First(&page)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &page, nil
}
