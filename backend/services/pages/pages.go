package pages

import (
	"blog-agent-go/backend/models"

	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) GetAboutPage() (*models.AboutPage, error) {
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

func (s *Service) GetContactPage() (*models.ContactPage, error) {
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
