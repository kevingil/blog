// Package openai provides a client for OpenAI API services
package openai

import (
	"context"
	"errors"

	openai "github.com/openai/openai-go"
)

const imagePromptSystem = "You are an image prompt generator. Given the content of an article, craft a vivid, concise prompt that an image generation model can use to create a representative illustration. Focus on key subjects, environment, style, mood, and colors. Respond with the prompt only."

// Client wraps the OpenAI client with convenience methods
type Client struct {
	client *openai.Client
}

// NewClient creates a new OpenAI client
func NewClient() *Client {
	c := openai.NewClient()
	return &Client{
		client: &c,
	}
}

// GenerateImagePrompt takes raw article text and returns a single prompt string suitable for the image generation model
func (c *Client) GenerateImagePrompt(ctx context.Context, articleText string) (string, error) {
	if articleText == "" {
		return "", errors.New("article text cannot be empty for prompt generation")
	}

	// Build chat completion request using official OpenAI client
	params := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(imagePromptSystem),
			openai.UserMessage(articleText),
		},
		Model: openai.ChatModel("gpt-5-2025-08-07"), // Updated to use GPT-5
	}

	completion, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return "", err
	}

	if len(completion.Choices) == 0 {
		return "", errors.New("no choices returned from openai chat completion")
	}

	return completion.Choices[0].Message.Content, nil
}

// GetRawClient returns the underlying OpenAI client for advanced usage
func (c *Client) GetRawClient() *openai.Client {
	return c.client
}
