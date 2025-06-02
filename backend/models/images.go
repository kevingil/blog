package models

type ImageGeneration struct {
	Prompt     string `json:"prompt"`
	Provider   string `json:"provider"`
	ModelName  string `json:"model"`
	RequestID  string `json:"request_id"`
	OutputURL  string `json:"output_url"`
	StorageKey string `json:"storage_key"`
	CreatedAt  int64  `json:"created_at"`
}
