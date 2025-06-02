package services

import (
	"database/sql"

	"blog-agent-go/backend/database"
	"blog-agent-go/backend/models"
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

	err := db.QueryRow("SELECT title, content, profile_image, meta_description, last_updated FROM about_page LIMIT 1").Scan(
		&page.Title, &page.Content, &page.ProfileImage, &page.MetaDescription, &page.LastUpdated,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &page, nil
}

func (s *PagesService) GetContactPage() (*models.ContactPage, error) {
	db := s.db.GetDB()
	var page models.ContactPage

	err := db.QueryRow("SELECT title, content, email_address, social_links, meta_description, last_updated FROM contact_page LIMIT 1").Scan(
		&page.Title, &page.Content, &page.EmailAddress, &page.SocialLinks, &page.MetaDescription, &page.LastUpdated,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &page, nil
}
