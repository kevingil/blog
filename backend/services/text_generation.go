package services

import (
	"context"
	"errors"

	openai "github.com/openai/openai-go"
)

const imagePromptSystem = "You are an image prompt generator. Given the content of an article, craft a vivid, concise prompt that an image generation model can use to create a representative illustration. Focus on key subjects, environment, style, mood, and colors. Respond with the prompt only."

// TextGenerationService wraps LLMService to expose higher-level helpers.
// For now it only exposes GenerateImagePrompt.
type TextGenerationService struct {
	client *openai.Client
}

func NewTextGenerationService() *TextGenerationService {
	c := openai.NewClient()
	return &TextGenerationService{
		client: &c,
	}
}

// GenerateImagePrompt takes raw article text and returns a single prompt string suitable for the image generation model.
func (t *TextGenerationService) GenerateImagePrompt(ctx context.Context, articleText string) (string, error) {
	if articleText == "" {
		return "", errors.New("article text cannot be empty for prompt generation")
	}

	// Build chat completion request using official OpenAI client
	params := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(imagePromptSystem),
			openai.UserMessage(articleText),
		},
		Model: openai.ChatModelGPT4o, // closest equivalent to previous gpt-4.1
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
