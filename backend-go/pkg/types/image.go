package types

import (
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

// Image status constants
const (
	ImageStatusPending   = "pending"
	ImageStatusCompleted = "completed"
	ImageStatusFailed    = "failed"
)
