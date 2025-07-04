package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"blog-agent-go/backend/database"
	"blog-agent-go/backend/models"

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

func (s *ImageGenerationService) GenerateArticleImage(ctx context.Context, prompt string, articleID int64, generatePrompt bool) (*models.ImageGeneration, error) {
	if articleID == 0 {
		return nil, nil
	}

	db := s.db.GetDB()

	// Instantiate helpers
	client := openai.NewClient()
	textGen := NewTextGenerationService()

	// Generate prompt from article content if requested or prompt empty
	if generatePrompt || prompt == "" {
		var articleContent string
		if err := db.QueryRow("SELECT content FROM articles WHERE id = ?", articleID).Scan(&articleContent); err != nil {
			return nil, err
		}

		generatedPrompt, err := textGen.GenerateImagePrompt(ctx, articleContent)
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
	key := fmt.Sprintf("images/articles/%d/%d.png", articleID, timestamp)
	if err := s.storage.UploadFile(ctx, key, imgBytes); err != nil {
		return nil, err
	}

	imageURL := fmt.Sprintf("%s/%s", os.Getenv("S3_URL_PREFIX"), key)

	// Update article with new image URL
	if _, err := db.Exec("UPDATE articles SET image = ?, image_generation_request_id = NULL WHERE id = ?", imageURL, articleID); err != nil {
		return nil, err
	}

	imageGen := &models.ImageGeneration{
		Prompt:    prompt,
		Provider:  "openai",
		ModelName: "gpt-image-1",
		OutputURL: imageURL,
		CreatedAt: timestamp,
	}

	return imageGen, nil
}

func (s *ImageGenerationService) GetImageGeneration(ctx context.Context, requestID string) (*models.ImageGeneration, error) {
	return nil, fmt.Errorf("GetImageGeneration no longer supported")
}

func (s *ImageGenerationService) GetImageGenerationStatus(ctx context.Context, requestID string) (*ImageGenerationStatus, error) {
	return &ImageGenerationStatus{Accepted: false, RequestID: requestID, OutputURL: ""}, nil
}
