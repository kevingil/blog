package models

import "time"

type AboutPage struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	Title           string    `json:"title" gorm:"not null"`
	Content         string    `json:"content" gorm:"type:text"`
	ProfileImage    string    `json:"profile_image"`
	MetaDescription string    `json:"meta_description"`
	LastUpdated     string    `json:"last_updated"`
}

type ContactPage struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	Title           string    `json:"title" gorm:"not null"`
	Content         string    `json:"content" gorm:"type:text"`
	EmailAddress    string    `json:"email_address" gorm:"not null"`
	SocialLinks     string    `json:"social_links" gorm:"type:text"`
	MetaDescription string    `json:"meta_description"`
	LastUpdated     string    `json:"last_updated"`
}

type Project struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Title       string    `json:"title" gorm:"not null"`
	Description string    `json:"description" gorm:"type:text"`
	URL         string    `json:"url"`
	Image       string    `json:"image"`
}
