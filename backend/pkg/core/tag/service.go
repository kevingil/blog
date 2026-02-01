package tag

import (
	"context"

	"backend/pkg/database/repository"

	"github.com/google/uuid"
)

// Service provides business logic for tags
type Service struct {
	repo repository.TagRepository
}

// NewService creates a new tag service
func NewService(repo repository.TagRepository) *Service {
	return &Service{repo: repo}
}

// GetByID retrieves a tag by its ID
func (s *Service) GetByID(ctx context.Context, id int) (*Tag, error) {
	return s.repo.FindByID(ctx, id)
}

// GetByName retrieves a tag by its name
func (s *Service) GetByName(ctx context.Context, name string) (*Tag, error) {
	return s.repo.FindByName(ctx, name)
}

// GetByIDs retrieves tags by their IDs
func (s *Service) GetByIDs(ctx context.Context, ids []int64) ([]Tag, error) {
	return s.repo.FindByIDs(ctx, ids)
}

// EnsureExists creates tags if they don't exist and returns their IDs
func (s *Service) EnsureExists(ctx context.Context, names []string) ([]int64, error) {
	return s.repo.EnsureExists(ctx, names)
}

// List retrieves all tags
func (s *Service) List(ctx context.Context) ([]Tag, error) {
	return s.repo.List(ctx)
}

// Create creates a new tag
func (s *Service) Create(ctx context.Context, name string) (*Tag, error) {
	tag := &Tag{
		Name: name,
	}

	if err := s.repo.Save(ctx, tag); err != nil {
		return nil, err
	}

	return tag, nil
}

// Delete removes a tag by its ID
func (s *Service) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}

// ResolveTagNames takes tag IDs and returns their names
func (s *Service) ResolveTagNames(ctx context.Context, ids []int64) ([]string, error) {
	if len(ids) == 0 {
		return []string{}, nil
	}

	tags, err := s.repo.FindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	names := make([]string, len(tags))
	for i, t := range tags {
		names[i] = t.Name
	}

	return names, nil
}

// Helper function to check if a tag is used (placeholder for actual implementation)
func (s *Service) IsTagUsed(ctx context.Context, id int) (bool, error) {
	// This would typically check if the tag is referenced by any articles or projects
	// For now, return false as a placeholder
	_ = uuid.Nil // Suppress unused import warning
	return false, nil
}
