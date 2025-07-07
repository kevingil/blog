package models

import "gorm.io/gorm"

type AboutPage struct {
	gorm.Model
	Title           string `json:"title" gorm:"not null"`
	Content         string `json:"content" gorm:"type:text"`
	ProfileImage    string `json:"profile_image"`
	MetaDescription string `json:"meta_description"`
	LastUpdated     string `json:"last_updated"`
}

type ContactPage struct {
	gorm.Model
	Title           string `json:"title" gorm:"not null"`
	Content         string `json:"content" gorm:"type:text"`
	EmailAddress    string `json:"email_address" gorm:"not null"`
	SocialLinks     string `json:"social_links" gorm:"type:text"`
	MetaDescription string `json:"meta_description"`
	LastUpdated     string `json:"last_updated"`
}

type Project struct {
	gorm.Model
	Title       string `json:"title" gorm:"not null"`
	Description string `json:"description" gorm:"type:text"`
	URL         string `json:"url"`
	Image       string `json:"image"`
}
