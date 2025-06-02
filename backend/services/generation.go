package services

import (
	"context"
	"time"

	"blog-agent-go/backend/database"
	"blog-agent-go/backend/models"
)

type ImageGenerationService struct {
	db database.Service
}

func NewImageGenerationService(db database.Service) *ImageGenerationService {
	return &ImageGenerationService{db: db}
}

type ImageGenerationStatus struct {
	Accepted  bool   `json:"accepted"`
	RequestID string `json:"request_id"`
	OutputURL string `json:"output_url"`
}

func (s *ImageGenerationService) GenerateArticleImage(ctx context.Context, prompt string, articleID int64, generatePrompt bool) (*models.ImageGeneration, error) {
	if articleID == 0 {
		return nil, nil
	}

	db := s.db.GetDB()

	// TODO: Implement image generation with external service
	imageGen := &models.ImageGeneration{
		Prompt:    prompt,
		Provider:  "fal",
		ModelName: "flux/dev",
		RequestID: "temp-request-id", // This will be replaced with actual request ID
		CreatedAt: time.Now().Unix(),
	}

	// Insert image generation record
	_, err := db.Exec("INSERT INTO image_generations (prompt, provider, model_name, request_id, created_at) VALUES (?, ?, ?, ?, ?)",
		imageGen.Prompt, imageGen.Provider, imageGen.ModelName, imageGen.RequestID, imageGen.CreatedAt)
	if err != nil {
		return nil, err
	}

	// Update article with image generation request ID
	_, err = db.Exec("UPDATE articles SET image_generation_request_id = ? WHERE id = ?", imageGen.RequestID, articleID)
	if err != nil {
		return nil, err
	}

	return imageGen, nil
}

func (s *ImageGenerationService) GetImageGeneration(ctx context.Context, requestID string) (*models.ImageGeneration, error) {
	db := s.db.GetDB()
	var imageGen models.ImageGeneration

	err := db.QueryRow("SELECT prompt, provider, model_name, request_id, output_url, storage_key, created_at FROM image_generations WHERE request_id = ?", requestID).Scan(
		&imageGen.Prompt, &imageGen.Provider, &imageGen.ModelName, &imageGen.RequestID, &imageGen.OutputURL, &imageGen.StorageKey, &imageGen.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &imageGen, nil
}

func (s *ImageGenerationService) GetImageGenerationStatus(ctx context.Context, requestID string) (*ImageGenerationStatus, error) {
	db := s.db.GetDB()

	// TODO: Implement status check with external service
	status := &ImageGenerationStatus{
		Accepted:  true,
		RequestID: requestID,
		OutputURL: "", // This will be populated with actual URL
	}

	// Update image generation record with output URL
	if status.OutputURL != "" {
		_, err := db.Exec("UPDATE image_generations SET output_url = ? WHERE request_id = ?", status.OutputURL, requestID)
		if err != nil {
			return nil, err
		}

		// Clear image generation request ID from article
		_, err = db.Exec("UPDATE articles SET image_generation_request_id = NULL WHERE image_generation_request_id = ?", requestID)
		if err != nil {
			return nil, err
		}
	}

	return status, nil
}
