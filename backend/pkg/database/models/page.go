package models

import (
	"encoding/json"
	"time"

	"backend/pkg/core/page"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Page is the GORM model for pages
type Page struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Slug        string         `json:"slug" gorm:"not null;uniqueIndex"`
	Title       string         `json:"title" gorm:"not null"`
	Content     string         `json:"content" gorm:"type:text"`
	Description string         `json:"description"`
	ImageURL    string         `json:"image_url"`
	MetaData    datatypes.JSON `json:"meta_data" gorm:"type:jsonb;default:'{}'"`
	IsPublished bool           `json:"is_published" gorm:"default:true"`
	CreatedAt   time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
}

func (Page) TableName() string {
	return "page"
}

// ToCore converts the GORM model to the domain type
func (m *Page) ToCore() *page.Page {
	var metaData map[string]interface{}
	if m.MetaData != nil {
		_ = json.Unmarshal(m.MetaData, &metaData)
	}

	return &page.Page{
		ID:          m.ID,
		Slug:        m.Slug,
		Title:       m.Title,
		Content:     m.Content,
		Description: m.Description,
		ImageURL:    m.ImageURL,
		MetaData:    metaData,
		IsPublished: m.IsPublished,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// PageFromCore creates a GORM model from the domain type
func PageFromCore(p *page.Page) *Page {
	var metaData datatypes.JSON
	if p.MetaData != nil {
		metaData, _ = datatypes.NewJSONType(p.MetaData).MarshalJSON()
	}

	return &Page{
		ID:          p.ID,
		Slug:        p.Slug,
		Title:       p.Title,
		Content:     p.Content,
		Description: p.Description,
		ImageURL:    p.ImageURL,
		MetaData:    metaData,
		IsPublished: p.IsPublished,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}
