package models

import (
	"time"
)

// Tag is the GORM model for tags
type Tag struct {
	ID        int       `json:"id" gorm:"primaryKey;autoIncrement"`
	Name      string    `json:"name" gorm:"uniqueIndex;not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (Tag) TableName() string {
	return "tag"
}
