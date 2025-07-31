package models

import (
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Title       string    `json:"title" gorm:"not null"`
	Description string    `json:"description" gorm:"type:text;not null"`
	ImageURL    string    `json:"image_url"`
	URL         string    `json:"url"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (Project) TableName() string {
	return "project"
}
