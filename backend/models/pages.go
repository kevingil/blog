package models

import (
	"gorm.io/gorm"
)

type AboutPage struct {
	gorm.Model
	Title           string `json:"title" gorm:"type:varchar(255);not null"`
	Content         string `json:"content" gorm:"type:text;not null"`
	ProfileImage    string `json:"profile_image" gorm:"type:varchar(255)"`
	MetaDescription string `json:"meta_description" gorm:"type:varchar(255)"`
	LastUpdated     string `json:"last_updated" gorm:"type:varchar(255)"`
}

type ContactPage struct {
	gorm.Model
	Title           string `json:"title" gorm:"type:varchar(255);not null"`
	Content         string `json:"content" gorm:"type:text;not null"`
	EmailAddress    string `json:"email_address" gorm:"type:varchar(255);not null"`
	SocialLinks     string `json:"social_links" gorm:"type:text"`
	MetaDescription string `json:"meta_description" gorm:"type:varchar(255)"`
	LastUpdated     string `json:"last_updated" gorm:"type:varchar(255)"`
}

type Project struct {
	gorm.Model
	Title       string `json:"title" gorm:"type:varchar(255);not null"`
	Description string `json:"description" gorm:"type:text;not null"`
	URL         string `json:"url" gorm:"type:varchar(255)"`
	Image       string `json:"image" gorm:"type:varchar(255)"`
}
