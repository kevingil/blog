// Package image provides the ImageGeneration domain type
package image

import "backend/pkg/types"

// ImageGeneration is an alias to types.ImageGeneration for backward compatibility
type ImageGeneration = types.ImageGeneration

// Status constants for image generation
const (
	StatusPending   = types.ImageStatusPending
	StatusCompleted = types.ImageStatusCompleted
	StatusFailed    = types.ImageStatusFailed
)
