package services

import (
	"backend/pkg/core"
	"backend/pkg/database"
	"backend/pkg/models"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// ProjectsService provides CRUD operations for Project model
type ProjectsService struct {
	db         database.Service
	tagService *TagService
}

func NewProjectsService(db database.Service) *ProjectsService {
	return &ProjectsService{
		db:         db,
		tagService: NewTagService(db),
	}
}

type ProjectCreateRequest struct {
	Title       string   `json:"title" validate:"required,min=3,max=200"`
	Description string   `json:"description" validate:"required,min=10,max=500"`
	Content     string   `json:"content"`
	Tags        []string `json:"tags" validate:"max=10,dive,min=2,max=30"`
	ImageURL    string   `json:"image_url" validate:"omitempty,url"`
	URL         string   `json:"url" validate:"omitempty,url"`
}

type ProjectUpdateRequest struct {
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	Content     *string    `json:"content"`
	Tags        *[]string  `json:"tags"`
	ImageURL    *string    `json:"image_url"`
	URL         *string    `json:"url"`
	CreatedAt   *time.Time `json:"created_at"`
}

func (s *ProjectsService) ListProjects(page int, perPage int) ([]models.Project, int64, error) {
	db := s.db.GetDB()

	if perPage <= 0 {
		perPage = 20
	}
	if page <= 0 {
		page = 1
	}

	var total int64
	if err := db.Model(&models.Project{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var projects []models.Project
	offset := (page - 1) * perPage
	if err := db.Order("created_at DESC").Offset(offset).Limit(perPage).Find(&projects).Error; err != nil {
		return nil, 0, err
	}
	return projects, total, nil
}

func (s *ProjectsService) GetProject(id uuid.UUID) (*models.Project, error) {
	db := s.db.GetDB()
	var project models.Project
	if err := db.First(&project, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.NotFoundError("Project")
		}
		return nil, core.InternalError("Failed to fetch project")
	}
	return &project, nil
}

func (s *ProjectsService) CreateProject(req ProjectCreateRequest) (*models.Project, error) {
	db := s.db.GetDB()
	if req.Title == "" || req.Description == "" {
		return nil, core.InvalidInputError("Title and description are required")
	}
	// Handle tags using tag service
	tagIDs, err := s.tagService.EnsureTagsExist(req.Tags)
	if err != nil {
		return nil, err
	}
	project := &models.Project{
		Title:       req.Title,
		Description: req.Description,
		Content:     req.Content,
		TagIDs:      pq.Int64Array(tagIDs),
		ImageURL:    req.ImageURL,
		URL:         req.URL,
	}
	if err := db.Create(project).Error; err != nil {
		return nil, core.InternalError("Failed to create project")
	}
	return project, nil
}

func (s *ProjectsService) UpdateProject(id uuid.UUID, req ProjectUpdateRequest) (*models.Project, error) {
	db := s.db.GetDB()
	var project models.Project
	if err := db.First(&project, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.NotFoundError("Project")
		}
		return nil, core.InternalError("Failed to fetch project")
	}
	if req.Title != nil {
		project.Title = *req.Title
	}
	if req.Description != nil {
		project.Description = *req.Description
	}
	if req.Content != nil {
		project.Content = *req.Content
	}
	if req.Tags != nil {
		tagIDs, err := s.tagService.EnsureTagsExist(*req.Tags)
		if err != nil {
			return nil, err
		}
		project.TagIDs = pq.Int64Array(tagIDs)
	}
	if req.ImageURL != nil {
		project.ImageURL = *req.ImageURL
	}
	if req.URL != nil {
		project.URL = *req.URL
	}
	if req.CreatedAt != nil {
		project.CreatedAt = *req.CreatedAt
	}
	if err := db.Save(&project).Error; err != nil {
		return nil, core.InternalError("Failed to update project")
	}
	return &project, nil
}

func (s *ProjectsService) DeleteProject(id uuid.UUID) error {
	db := s.db.GetDB()
	result := db.Delete(&models.Project{}, "id = ?", id)
	if result.Error != nil {
		return core.InternalError("Failed to delete project")
	}
	if result.RowsAffected == 0 {
		return core.NotFoundError("Project")
	}
	return nil
}

// ProjectDetail includes project with resolved tag names
type ProjectDetail struct {
	Project models.Project `json:"project"`
	Tags    []string       `json:"tags"`
}

func (s *ProjectsService) GetProjectDetail(id uuid.UUID) (*ProjectDetail, error) {
	db := s.db.GetDB()
	var project models.Project
	if err := db.First(&project, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.NotFoundError("Project")
		}
		return nil, core.InternalError("Failed to fetch project")
	}
	var names []string
	if len(project.TagIDs) > 0 {
		var tagModels []models.Tag
		if err := db.Where("id IN ?", project.TagIDs).Find(&tagModels).Error; err == nil {
			for _, t := range tagModels {
				names = append(names, t.Name)
			}
		}
	}
	return &ProjectDetail{Project: project, Tags: names}, nil
}
