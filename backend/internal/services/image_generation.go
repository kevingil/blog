package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"blog-agent-go/backend/internal/database"
	"blog-agent-go/backend/internal/models"

	"github.com/google/uuid"
	"github.com/openai/openai-go"
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
			return nil, result.Error
		}

		generatedPrompt, err := textGen.GenerateImagePrompt(ctx, article.Content)
		if err != nil {
			return nil, err
		}
		prompt = generatedPrompt
	}

	// Generate image using official OpenAI client and get base64 response
	imgResp, err := client.Images.Generate(ctx, openai.ImageGenerateParams{
		Prompt:         prompt,
		Model:          openai.ImageModelGPTImage1,
		ResponseFormat: openai.ImageGenerateParamsResponseFormatB64JSON,
		N:              openai.Int(1),
	})
	if err != nil {
		return nil, err
	}

	if len(imgResp.Data) == 0 {
		return nil, fmt.Errorf("no data returned from openai image generation")
	}

	b64Img := imgResp.Data[0].B64JSON

	// Decode base64
	imgBytes, err := base64.StdEncoding.DecodeString(b64Img)
	if err != nil {
		return nil, err
	}

	// Build storage key and upload
	timestamp := time.Now().Unix()
	key := fmt.Sprintf("images/articles/%s/%d.png", articleID, timestamp)
	if err := s.storage.UploadFile(ctx, key, imgBytes); err != nil {
		return nil, err
	}

	imageURL := fmt.Sprintf("%s/%s", os.Getenv("S3_URL_PREFIX"), key)

	// Update article with new image URL
	result := db.Model(&models.Article{}).Where("id = ?", articleID).Updates(map[string]interface{}{
		"image_url":         imageURL,
		"imagen_request_id": nil,
	})
	if result.Error != nil {
		return nil, result.Error
	}

	imageGen := &models.ImageGeneration{
		Prompt:    prompt,
		Provider:  "openai",
		ModelName: "gpt-image-1",
		OutputURL: imageURL,
	}

	// Save to database
	result = db.Create(imageGen)
	if result.Error != nil {
		return nil, result.Error
	}

	return imageGen, nil
}

func (s *ImageGenerationService) GetImageGeneration(ctx context.Context, requestID string) (*models.ImageGeneration, error) {
	db := s.db.GetDB()
	var imageGen models.ImageGeneration

	result := db.Where("request_id = ?", requestID).First(&imageGen)
	if result.Error != nil {
		return nil, result.Error
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
