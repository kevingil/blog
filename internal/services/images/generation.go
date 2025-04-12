package images

import (
	"context"
	"time"

	"blog-agent-go/internal/models"

	"gorm.io/gorm"
)

type ImageGenerationService struct {
	db *gorm.DB
}

func NewImageGenerationService(db *gorm.DB) *ImageGenerationService {
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

	// TODO: Implement image generation with external service
	imageGen := &models.ImageGeneration{
		Prompt:    prompt,
		Provider:  "fal",
		ModelName: "flux/dev",
		RequestID: "temp-request-id", // This will be replaced with actual request ID
		CreatedAt: time.Now().Unix(),
	}

	if err := s.db.Create(imageGen).Error; err != nil {
		return nil, err
	}

	// Update article with image generation request ID
	if err := s.db.Model(&models.Article{}).Where("id = ?", articleID).Update("image_generation_request_id", imageGen.RequestID).Error; err != nil {
		return nil, err
	}

	return imageGen, nil
}

func (s *ImageGenerationService) GetImageGeneration(ctx context.Context, requestID string) (*models.ImageGeneration, error) {
	var imageGen models.ImageGeneration
	if err := s.db.Where("request_id = ?", requestID).First(&imageGen).Error; err != nil {
		return nil, err
	}
	return &imageGen, nil
}

func (s *ImageGenerationService) GetImageGenerationStatus(ctx context.Context, requestID string) (*ImageGenerationStatus, error) {
	// TODO: Implement status check with external service
	status := &ImageGenerationStatus{
		Accepted:  true,
		RequestID: requestID,
		OutputURL: "", // This will be populated with actual URL
	}

	// Update image generation record with output URL
	if status.OutputURL != "" {
		if err := s.db.Model(&models.ImageGeneration{}).Where("request_id = ?", requestID).Update("output_url", status.OutputURL).Error; err != nil {
			return nil, err
		}

		// Clear image generation request ID from article
		if err := s.db.Model(&models.Article{}).Where("image_generation_request_id = ?", requestID).Update("image_generation_request_id", nil).Error; err != nil {
			return nil, err
		}
	}

	return status, nil
}
