package models

import "time"

type ImageGeneration struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Prompt     string    `json:"prompt" gorm:"not null"`
	Provider   string    `json:"provider" gorm:"not null"`
	ModelName  string    `json:"model" gorm:"not null"`
	RequestID  string    `json:"request_id" gorm:"uniqueIndex;not null"`
	OutputURL  string    `json:"output_url"`
	StorageKey string    `json:"storage_key"`
}
