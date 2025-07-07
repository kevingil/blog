package models

import "gorm.io/gorm"

type ImageGeneration struct {
	gorm.Model
	Prompt     string `json:"prompt" gorm:"not null"`
	Provider   string `json:"provider" gorm:"not null"`
	ModelName  string `json:"model" gorm:"not null"`
	RequestID  string `json:"request_id" gorm:"uniqueIndex;not null"`
	OutputURL  string `json:"output_url"`
	StorageKey string `json:"storage_key"`
}
