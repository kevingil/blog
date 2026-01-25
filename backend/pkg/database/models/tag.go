package models

import (
	"time"

	"backend/pkg/core/tag"
)

// TagModel is the GORM model for tags
type TagModel struct {
	ID        int       `json:"id" gorm:"primaryKey;autoIncrement"`
	Name      string    `json:"name" gorm:"uniqueIndex;not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (TagModel) TableName() string {
	return "tag"
}

// ToCore converts the GORM model to the domain type
func (m *TagModel) ToCore() *tag.Tag {
	return &tag.Tag{
		ID:        m.ID,
		Name:      m.Name,
		CreatedAt: m.CreatedAt,
	}
}

// TagModelFromCore creates a GORM model from the domain type
func TagModelFromCore(t *tag.Tag) *TagModel {
	return &TagModel{
		ID:        t.ID,
		Name:      t.Name,
		CreatedAt: t.CreatedAt,
	}
}
