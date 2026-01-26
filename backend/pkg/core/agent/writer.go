package agent

import (
	"context"
	"fmt"
	"log"
	"os"

	"backend/pkg/core/agent/prompts"
	"backend/pkg/core/ml"
	"backend/pkg/database/models"

	"github.com/google/uuid"
	openai "github.com/openai/openai-go"
)

// WriterAgent provides article generation using LLM with writer and editor prompts
type WriterAgent struct {
	client           *openai.Client
	embeddingService *ml.EmbeddingService
}

// NewWriterAgent creates a new WriterAgent instance
func NewWriterAgent() *WriterAgent {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}
	client := openai.NewClient()
	return &WriterAgent{
		client:           &client,
		embeddingService: ml.NewEmbeddingService(),
	}
}

// GenerateArticle creates a new article using the writer and editor prompts
func (w *WriterAgent) GenerateArticle(ctx context.Context, prompt, title string, authorID uuid.UUID) (*models.Article, error) {
	// First draft with writer system message
	draftMsg, err := w.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(prompts.WriterSystemPrompt(prompts.WritingContext)),
			openai.UserMessage(prompts.WriterUserPrompt(title, prompt)),
		},
		Model: openai.ChatModelGPT4o,
	})
	if err != nil {
		return nil, fmt.Errorf("error generating draft: %w", err)
	}

	// Editor refinement
	finalMsg, err := w.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(prompts.EditorSystemPrompt),
			openai.UserMessage(draftMsg.Choices[0].Message.Content),
		},
		Model: openai.ChatModelGPT4o,
	})
	if err != nil {
		return nil, fmt.Errorf("error refining article: %w", err)
	}

	// Generate embedding for the content
	embedding, err := w.embeddingService.GenerateEmbedding(ctx, finalMsg.Choices[0].Message.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Create article with draft content
	article := &models.Article{
		DraftTitle:     title,
		DraftContent:   finalMsg.Choices[0].Message.Content,
		AuthorID:       authorID,
		DraftEmbedding: embedding,
	}
	return article, nil
}

// UpdateWithContext updates an article using editor context prompts
func (w *WriterAgent) UpdateWithContext(ctx context.Context, article *models.Article) (string, error) {
	if article == nil {
		return "", fmt.Errorf("article not found")
	}

	msg, err := w.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(prompts.EditorContextPrompt),
			openai.UserMessage(fmt.Sprintf("Title: %q\nPrompt: %s", article.DraftTitle, article.DraftContent)),
		},
		Model: openai.ChatModelGPT4o,
	})
	if err != nil {
		return "", fmt.Errorf("error updating article: %w", err)
	}

	return msg.Choices[0].Message.Content, nil
}
