package services

import (
	"backend/pkg/database"
	"backend/pkg/models"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// TagService provides reusable tag handling functions for managing article and project tags.
type TagService struct {
	db database.Service
}

// NewTagService creates a new tag service instance.
func NewTagService(db database.Service) *TagService {
	return &TagService{db: db}
}

// EnsureTagsExist ensures that all tags exist in the database, creating them if necessary
// Returns the tag IDs for all provided tag names
func (s *TagService) EnsureTagsExist(tagNames []string) ([]int64, error) {
	db := s.db.GetDB()
	var tagIDs []int64

	for _, tagName := range tagNames {
		tagName = strings.ToLower(strings.TrimSpace(tagName))
		if tagName == "" {
			continue
		}

		var tag models.Tag
		result := db.Where("LOWER(name) = ?", tagName).First(&tag)

		if result.Error == gorm.ErrRecordNotFound {
			// Create the tag if it doesn't exist
			tag = models.Tag{Name: tagName}
			if err := db.Create(&tag).Error; err != nil {
				return nil, fmt.Errorf("failed to create tag '%s': %w", tagName, err)
			}
		} else if result.Error != nil {
			return nil, fmt.Errorf("failed to check tag existence: %w", result.Error)
		}

		tagIDs = append(tagIDs, int64(tag.ID))
	}

	return tagIDs, nil
}

// GetTagsByIDs retrieves tags by their IDs.
// Returns an empty slice if no IDs are provided.
func (s *TagService) GetTagsByIDs(ids []int64) ([]models.Tag, error) {
	if len(ids) == 0 {
		return []models.Tag{}, nil
	}

	db := s.db.GetDB()
	var tags []models.Tag

	if err := db.Where("id IN ?", ids).Find(&tags).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch tags: %w", err)
	}

	return tags, nil
}

// GetTagByName retrieves a tag by its name (case-insensitive).
// Returns nil if the tag is not found.
func (s *TagService) GetTagByName(name string) (*models.Tag, error) {
	db := s.db.GetDB()
	var tag models.Tag

	name = strings.ToLower(strings.TrimSpace(name))
	result := db.Where("LOWER(name) = ?", name).First(&tag)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch tag: %w", result.Error)
	}

	return &tag, nil
}
