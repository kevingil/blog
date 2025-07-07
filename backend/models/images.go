package models

type ImageGeneration struct {
	ID         uint   `json:"id" gorm:"primaryKey"`
	CreatedAt  int64  `json:"created_at" gorm:"autoCreateTime"`
	Prompt     string `json:"prompt" gorm:"not null"`
	Provider   string `json:"provider" gorm:"not null"`
	ModelName  string `json:"model" gorm:"not null;column:model"`
	RequestID  string `json:"request_id" gorm:"uniqueIndex;not null"`
	OutputURL  string `json:"output_url"`
	StorageKey string `json:"storage_key"`
}

func (ImageGeneration) TableName() string {
	return "image_generation"
}
