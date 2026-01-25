package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"backend/pkg/core"
	"backend/pkg/database"
	"backend/pkg/models"

	"github.com/google/uuid"
	"github.com/openai/openai-go"
	"gorm.io/gorm"
)

type ImageGenerationService struct {
	db      database.Service
	storage *StorageService
}

func NewImageGenerationService(db database.Service, storage *StorageService) *ImageGenerationService {
	return &ImageGenerationService{db: db, storage: storage}
}

type ImageGenerationStatus struct {
	Accepted  bool   `json:"accepted"`
	RequestID string `json:"request_id"`
	OutputURL string `json:"output_url"`
}

func (s *ImageGenerationService) GenerateArticleImage(ctx context.Context, prompt string, articleID uuid.UUID, generatePrompt bool) (*models.ImageGeneration, error) {
	if articleID == uuid.Nil {
		return nil, nil
	}

	db := s.db.GetDB()

	// Instantiate helpers
	client := openai.NewClient()
	textGen := NewTextGenerationService()

	// Generate prompt from article content if requested or prompt empty
	if generatePrompt || prompt == "" {
		var article models.Article
		result := db.Select("content").First(&article, articleID)
		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				return nil, core.NotFoundError("Article")
			}
			return nil, core.InternalError("Failed to fetch article")
		}

		generatedPrompt, err := textGen.GenerateImagePrompt(ctx, article.DraftContent)
		if err != nil {
			return nil, core.InternalError("Failed to generate image prompt")
		}
		prompt = generatedPrompt
	}

	// Generate image using official OpenAI client and get base64 response
	imgResp, err := client.Images.Generate(ctx, openai.ImageGenerateParams{
		Prompt:       prompt,
		Model:        openai.ImageModelGPTImage1,
		N:            openai.Int(1),
		Size:         openai.ImageGenerateParamsSizeAuto,        // Auto selects best size for gpt-image-1
		Quality:      openai.ImageGenerateParamsQualityHigh,     // High quality for better article images
		OutputFormat: openai.ImageGenerateParamsOutputFormatPNG, // PNG for better quality
		Background:   openai.ImageGenerateParamsBackgroundAuto,  // Auto background selection
		Moderation:   openai.ImageGenerateParamsModerationAuto,  // Default content moderation
	})
	if err != nil {
		return nil, core.InternalError("Failed to generate image from OpenAI")
	}

	if len(imgResp.Data) == 0 {
		return nil, core.InternalError("No data returned from OpenAI image generation")
	}

	b64Img := imgResp.Data[0].B64JSON

	// Decode base64
	imgBytes, err := base64.StdEncoding.DecodeString(b64Img)
	if err != nil {
		return nil, core.InternalError("Failed to decode image data")
	}

	// Build storage key and upload
	timestamp := time.Now().Unix()
	key := fmt.Sprintf("images/articles/%s/%d.png", articleID, timestamp)
	if err := s.storage.UploadFile(ctx, key, imgBytes); err != nil {
		return nil, core.InternalError("Failed to upload image to storage")
	}

	imageURL := fmt.Sprintf("%s/%s", os.Getenv("S3_URL_PREFIX"), key)

	// Update article with new image URL
	result := db.Model(&models.Article{}).Where("id = ?", articleID).Updates(map[string]interface{}{
		"image_url":         imageURL,
		"imagen_request_id": nil,
	})
	if result.Error != nil {
		return nil, core.InternalError("Failed to update article with image URL")
	}

	imageGen := &models.ImageGeneration{
		Prompt:    prompt,
		Provider:  "openai",
		ModelName: "gpt-image-1",
		OutputURL: imageURL,
		RequestID: uuid.New().String(),
		Status:    "completed",
	}

	// Save to database
	result = db.Create(imageGen)
	if result.Error != nil {
		return nil, core.InternalError("Failed to save image generation record")
	}

	return imageGen, nil
}

func (s *ImageGenerationService) GetImageGeneration(ctx context.Context, requestID string) (*models.ImageGeneration, error) {
	db := s.db.GetDB()
	var imageGen models.ImageGeneration

	result := db.Where("request_id = ?", requestID).First(&imageGen)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, core.NotFoundError("Image generation")
		}
		return nil, core.InternalError("Failed to fetch image generation")
	}

	return &imageGen, nil
}

func (s *ImageGenerationService) GetImageGenerationStatus(ctx context.Context, requestID string) (*ImageGenerationStatus, error) {
	imageGen, err := s.GetImageGeneration(ctx, requestID)
	if err != nil {
		return &ImageGenerationStatus{Accepted: false, RequestID: requestID, OutputURL: ""}, nil
	}

	return &ImageGenerationStatus{
		Accepted:  true,
		RequestID: requestID,
		OutputURL: imageGen.OutputURL,
	}, nil
}
