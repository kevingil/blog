package image

import (
	"context"
	"time"

	"backend/pkg/database/repository"

	"github.com/google/uuid"
)

// Service provides business logic for image generations
type Service struct {
	repo repository.ImageRepository
}

// NewService creates a new image service
func NewService(repo repository.ImageRepository) *Service {
	return &Service{repo: repo}
}

// CreateRequest represents a request to create an image generation
type CreateRequest struct {
	Prompt    string
	Provider  string
	ModelName string
	RequestID string
	MetaData  map[string]interface{}
}

// GetByID retrieves an image generation by its ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*ImageGeneration, error) {
	return s.repo.FindByID(ctx, id)
}

// GetByRequestID retrieves an image generation by its request ID
func (s *Service) GetByRequestID(ctx context.Context, requestID string) (*ImageGeneration, error) {
	return s.repo.FindByRequestID(ctx, requestID)
}

// Create creates a new image generation record
func (s *Service) Create(ctx context.Context, req CreateRequest) (*ImageGeneration, error) {
	img := &ImageGeneration{
		ID:        uuid.New(),
		Prompt:    req.Prompt,
		Provider:  req.Provider,
		ModelName: req.ModelName,
		RequestID: req.RequestID,
		Status:    StatusPending,
		MetaData:  req.MetaData,
	}

	if err := s.repo.Save(ctx, img); err != nil {
		return nil, err
	}

	return img, nil
}

// MarkCompleted marks an image generation as completed
func (s *Service) MarkCompleted(ctx context.Context, id uuid.UUID, outputURL string, fileIndexID *uuid.UUID) error {
	img, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now()
	img.Status = StatusCompleted
	img.OutputURL = outputURL
	img.FileIndexID = fileIndexID
	img.CompletedAt = &now

	return s.repo.Update(ctx, img)
}

// MarkFailed marks an image generation as failed
func (s *Service) MarkFailed(ctx context.Context, id uuid.UUID, errorMessage string) error {
	img, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now()
	img.Status = StatusFailed
	img.ErrorMessage = errorMessage
	img.CompletedAt = &now

	return s.repo.Update(ctx, img)
}

// GetStatus returns the status of an image generation
func (s *Service) GetStatus(ctx context.Context, id uuid.UUID) (string, error) {
	img, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return "", err
	}
	return img.Status, nil
}
