// Package image provides the ImageGeneration domain type and store interface
package image

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ImageGeneration represents an image generation request
type ImageGeneration struct {
	ID           uuid.UUID
	Prompt       string
	Provider     string
	ModelName    string
	RequestID    string
	Status       string
	OutputURL    string
	FileIndexID  *uuid.UUID
	ErrorMessage string
	MetaData     map[string]interface{}
	CreatedAt    time.Time
	CompletedAt  *time.Time
}

// Status constants for image generation
const (
	StatusPending   = "pending"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)

// ImageStore defines the data access interface for image generations
type ImageStore interface {
	FindByID(ctx context.Context, id uuid.UUID) (*ImageGeneration, error)
	FindByRequestID(ctx context.Context, requestID string) (*ImageGeneration, error)
	Save(ctx context.Context, img *ImageGeneration) error
	Update(ctx context.Context, img *ImageGeneration) error
}
