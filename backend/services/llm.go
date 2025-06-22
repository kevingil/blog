package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	openAIBaseURL              = "https://api.openai.com/v1"
	defaultOpenAITimeout       = 60 * time.Second
	responseFormatJSON         = "json_object"
	responseFormatText         = "text"
	imageGenerationModel       = "gpt-image-1"
	defaultImageResponseFormat = "url" // could also be b64_json
)

// ChatMessage represents a single message in the chat conversation.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content,omitempty"`
	Name    string `json:"name,omitempty"`
}

// ChatCompletionRequest mirrors the OpenAI API request payload.
type ChatCompletionRequest struct {
	Model          string            `json:"model"`
	Messages       []ChatMessage     `json:"messages"`
	Tools          []map[string]any  `json:"tools,omitempty"`
	ResponseFormat map[string]string `json:"response_format,omitempty"`
}

// ChatCompletionChoice is a single choice in the chat completion response.
type ChatCompletionChoice struct {
	Index   int         `json:"index"`
	Message ChatMessage `json:"message"`
}

// ChatCompletionResponse represents the minimal structure we care about.
type ChatCompletionResponse struct {
	Choices []ChatCompletionChoice `json:"choices"`
}

// ImageGenerationRequest mirrors the OpenAI image generation payload.
type ImageGenerationRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	Size           string `json:"size,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
}

// imageData is the structure of each item in the images response.
type imageData struct {
	URL     string `json:"url,omitempty"`
	B64JSON string `json:"b64_json,omitempty"`
}

// ImageGenerationResponse is the subset of the API response we need.
type ImageGenerationResponse struct {
	Created int64       `json:"created"`
	Data    []imageData `json:"data"`
}

// LLMService provides helper methods for interacting with OpenAI APIs.
type LLMService struct {
	apiKey string
	client *http.Client
}

// NewLLMService returns a configured LLMService. If OPENAI_API_KEY is not set
// it will panic, as continuing without an API key would always fail.
func NewLLMService() *LLMService {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		panic("OPENAI_API_KEY environment variable not set")
	}

	return &LLMService{
		apiKey: apiKey,
		client: &http.Client{Timeout: defaultOpenAITimeout},
	}
}

// ChatCompletion performs a chat completion call with arbitrary messages/tools/format.
// If responseFormat is empty the default (text) is used.
func (s *LLMService) ChatCompletion(ctx context.Context, model string, messages []ChatMessage, tools []map[string]any, responseFormat string) (string, error) {
	if model == "" {
		return "", errors.New("model is required")
	}

	if responseFormat == "" {
		responseFormat = responseFormatText
	}

	payload := ChatCompletionRequest{
		Model:    model,
		Messages: messages,
	}

	if len(tools) > 0 {
		payload.Tools = tools
	}

	// Only include response format when the caller explicitly wants JSON.
	if responseFormat == responseFormatJSON {
		payload.ResponseFormat = map[string]string{"type": responseFormatJSON}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	endpoint := fmt.Sprintf("%s/chat/completions", openAIBaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("openai chat completion returned status %d", resp.StatusCode)
	}

	var completion ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&completion); err != nil {
		return "", err
	}

	if len(completion.Choices) == 0 {
		return "", errors.New("no choices returned from openai chat completion")
	}

	return completion.Choices[0].Message.Content, nil
}

// GenerateImage creates an image with the given prompt and returns a URL (or base64 string if requested).
func (s *LLMService) GenerateImage(ctx context.Context, prompt string, model string, size string, responseFormat string) (string, error) {
	if prompt == "" {
		return "", errors.New("prompt is required")
	}

	if model == "" {
		model = imageGenerationModel
	}

	if responseFormat == "" {
		responseFormat = defaultImageResponseFormat
	}

	reqPayload := ImageGenerationRequest{
		Model:          model,
		Prompt:         prompt,
		ResponseFormat: responseFormat,
	}

	if size != "" {
		reqPayload.Size = size
	}

	body, err := json.Marshal(reqPayload)
	if err != nil {
		return "", err
	}

	endpoint := fmt.Sprintf("%s/images/generations", openAIBaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("openai image generation returned status %d", resp.StatusCode)
	}

	var imgResp ImageGenerationResponse
	if err := json.NewDecoder(resp.Body).Decode(&imgResp); err != nil {
		return "", err
	}

	if len(imgResp.Data) == 0 {
		return "", errors.New("no data returned from openai image generation")
	}

	// Prefer URL format. If caller requested b64_json they should handle decoding themselves.
	if imgResp.Data[0].URL != "" {
		return imgResp.Data[0].URL, nil
	}

	if imgResp.Data[0].B64JSON != "" {
		return imgResp.Data[0].B64JSON, nil
	}

	return "", errors.New("image generation response contained no usable data")
}
