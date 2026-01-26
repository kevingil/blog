// Package image provides the ImageGeneration domain type and store interface
package image

import (
	"context"

	"backend/pkg/types"

	"github.com/google/uuid"
)

// ImageGeneration is an alias to types.ImageGeneration for backward compatibility
type ImageGeneration = types.ImageGeneration

// Status constants for image generation
const (
	StatusPending   = types.ImageStatusPending
	StatusCompleted = types.ImageStatusCompleted
	StatusFailed    = types.ImageStatusFailed
)

// ImageStore defines the data access interface for image generations
type ImageStore interface {
	FindByID(ctx context.Context, id uuid.UUID) (*types.ImageGeneration, error)
	FindByRequestID(ctx context.Context, requestID string) (*types.ImageGeneration, error)
	Save(ctx context.Context, img *types.ImageGeneration) error
	Update(ctx context.Context, img *types.ImageGeneration) error
}
