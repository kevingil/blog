package models

import (
	"gorm.io/gorm"
)

type ImageGeneration struct {
	gorm.Model
	Prompt     string `json:"prompt" gorm:"type:text;not null"`
	Provider   string `json:"provider" gorm:"type:varchar(255);not null"`
	ModelName  string `json:"model" gorm:"type:varchar(255);not null"`
	RequestID  string `json:"request_id" gorm:"type:varchar(255);not null;unique"`
	OutputURL  string `json:"output_url" gorm:"type:text"`
	StorageKey string `json:"storage_key" gorm:"type:varchar(255)"`
	CreatedAt  int64  `json:"created_at" gorm:"type:bigint"`
}
