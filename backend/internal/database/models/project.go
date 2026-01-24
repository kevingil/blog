package models

import (
	"time"

	"blog-agent-go/backend/internal/core/project"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ProjectModel is the GORM model for projects
type ProjectModel struct {
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

func (ProjectModel) TableName() string {
	return "project"
}

// ToCore converts the GORM model to the domain type
func (m *ProjectModel) ToCore() *project.Project {
	return &project.Project{
		ID:          m.ID,
		Title:       m.Title,
		Description: m.Description,
		Content:     m.Content,
		TagIDs:      m.TagIDs,
		ImageURL:    m.ImageURL,
		URL:         m.URL,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// ProjectModelFromCore creates a GORM model from the domain type
func ProjectModelFromCore(p *project.Project) *ProjectModel {
	return &ProjectModel{
		ID:          p.ID,
		Title:       p.Title,
		Description: p.Description,
		Content:     p.Content,
		TagIDs:      p.TagIDs,
		ImageURL:    p.ImageURL,
		URL:         p.URL,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}
