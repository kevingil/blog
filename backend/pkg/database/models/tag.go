package models

import (
	"time"

	"backend/pkg/core/tag"
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

// ToCore converts the GORM model to the domain type
func (m *Tag) ToCore() *tag.Tag {
	return &tag.Tag{
		ID:        m.ID,
		Name:      m.Name,
		CreatedAt: m.CreatedAt,
	}
}

// TagFromCore creates a GORM model from the domain type
func TagFromCore(t *tag.Tag) *Tag {
	return &Tag{
		ID:        t.ID,
		Name:      t.Name,
		CreatedAt: t.CreatedAt,
	}
}
