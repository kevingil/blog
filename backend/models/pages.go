package models

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Page struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Slug        string         `json:"slug" gorm:"not null;uniqueIndex"`
	Title       string         `json:"title" gorm:"not null"`
	Content     string         `json:"content" gorm:"type:text"`
	Description string         `json:"description"`
	ImageURL    string         `json:"image_url"`
	MetaData    datatypes.JSON `json:"meta_data" gorm:"type:jsonb;default:'{}'"`
	IsPublished bool           `json:"is_published" gorm:"default:true"`
	CreatedAt   string         `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   string         `json:"updated_at" gorm:"autoUpdateTime"`
}

func (Page) TableName() string {
	return "page"
}

type Project struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	Title       string `json:"title" gorm:"not null"`
	Description string `json:"description" gorm:"type:text"`
	URL         string `json:"url"`
	Image       string `json:"image"`
}
