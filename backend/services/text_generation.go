package services

import (
	"context"
	"errors"
)

const imagePromptSystem = "You are an image prompt generator. Given the content of an article, craft a vivid, concise prompt that an image generation model can use to create a representative illustration. Focus on key subjects, environment, style, mood, and colors. Respond with the prompt only."

// TextGenerationService wraps LLMService to expose higher-level helpers.
// For now it only exposes GenerateImagePrompt.
type TextGenerationService struct {
	llm *LLMService
}

func NewTextGenerationService() *TextGenerationService {
	return &TextGenerationService{
		llm: NewLLMService(),
	}
}

// GenerateImagePrompt takes raw article text and returns a single prompt string suitable for the image generation model.
func (t *TextGenerationService) GenerateImagePrompt(ctx context.Context, articleText string) (string, error) {
	if articleText == "" {
		return "", errors.New("article text cannot be empty for prompt generation")
	}

	messages := []ChatMessage{
		{Role: "system", Content: imagePromptSystem},
		{Role: "user", Content: articleText},
	}

	prompt, err := t.llm.ChatCompletion(ctx, "gpt-4.1", messages, nil, "")
	if err != nil {
		return "", err
	}

	return prompt, nil
}
