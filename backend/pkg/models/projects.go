package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Project struct {
	ID          uuid.UUID     `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Title       string        `json:"title" gorm:"not null"`
	Description string        `json:"description" gorm:"type:text;not null"`
	Content     string        `json:"content" gorm:"type:text"`
	TagIDs      pq.Int64Array `json:"tag_ids" gorm:"type:integer[]"`
	ImageURL    string        `json:"image_url"`
	URL         string        `json:"url"`
	CreatedAt   time.Time     `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time     `json:"updated_at" gorm:"autoUpdateTime"`
}

func (Project) TableName() string {
	return "project"
}
