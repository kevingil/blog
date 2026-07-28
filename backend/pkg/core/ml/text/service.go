// Package text provides text generation utilities using LLM
package text

import (
	"context"
	"errors"

	openai "github.com/openai/openai-go"
)

const imagePromptSystem = "You are an image prompt generator. Given the content of an article, craft a vivid, concise prompt that an image generation model can use to create a representative illustration. Focus on key subjects, environment, style, mood, and colors. Respond with the prompt only."

// GenerationService provides text generation utilities using LLM
type GenerationService struct {
	client *openai.Client
}

// NewGenerationService creates a new text generation service
func NewGenerationService() *GenerationService {
	c := openai.NewClient()
	return &GenerationService{
		client: &c,
	}
}

// GenerateImagePrompt takes raw article text and returns a single prompt string suitable for the image generation model
func (t *GenerationService) GenerateImagePrompt(ctx context.Context, articleText string) (string, error) {
	if articleText == "" {
		return "", errors.New("article text cannot be empty for prompt generation")
	}

	// Build chat completion request using official OpenAI client
	params := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(imagePromptSystem),
			openai.UserMessage(articleText),
		},
		Model: openai.ChatModel("gpt-5-2025-08-07"),
	}

	completion, err := t.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return "", err
	}

	if len(completion.Choices) == 0 {
		return "", errors.New("no choices returned from openai chat completion")
	}

	return completion.Choices[0].Message.Content, nil
}
