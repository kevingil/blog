package services

import (
	"blog-agent-go/backend/database"
	"blog-agent-go/backend/models"

	"gorm.io/gorm"
)

type PagesService struct {
	db database.Service
}

func NewPagesService(db database.Service) *PagesService {
	return &PagesService{db: db}
}

func (s *PagesService) GetAboutPage() (*models.AboutPage, error) {
	db := s.db.GetDB()
	var page models.AboutPage

	result := db.First(&page)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &page, nil
}

func (s *PagesService) GetContactPage() (*models.ContactPage, error) {
	db := s.db.GetDB()
	var page models.ContactPage

	result := db.First(&page)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &page, nil
}
