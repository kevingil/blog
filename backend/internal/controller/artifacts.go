package controller

import (
	"blog-agent-go/backend/internal/core/chat"
	"blog-agent-go/backend/internal/errors"
	"blog-agent-go/backend/internal/response"
	"blog-agent-go/backend/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// AcceptArtifactHandler handles accepting an artifact
func AcceptArtifactHandler(chatService *chat.MessageService, articleService *services.ArticleService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		messageIDStr := c.Params("messageId")
		messageID, err := uuid.Parse(messageIDStr)
		if err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid message ID"))
		}

		var req struct {
			Feedback string `json:"feedback"`
		}
		if err := c.BodyParser(&req); err != nil {
			// Feedback is optional, so body parse errors are okay
			req.Feedback = ""
		}

		// Accept the artifact
		if err := chatService.AcceptArtifact(c.Context(), messageID, req.Feedback); err != nil {
			return response.Error(c, err)
		}

		// Get artifact content and apply it to the article
		artifactContent, err := chatService.GetArtifactContent(c.Context(), messageID)
		if err != nil {
			return response.Error(c, err)
		}

		// Get current article to preserve fields
		message, err := chatService.GetMessageByID(c.Context(), messageID)
		if err != nil {
			return response.Error(c, err)
		}

		// Update article with artifact content
		// This is a simplified implementation - you may want to handle different artifact types differently
		updateReq := services.ArticleUpdateRequest{
			Content: artifactContent,
		}
		
		// Get current article to preserve other fields
		// For simplicity, we're just updating the content field
		// You may want to add more sophisticated merging logic
		_, err = articleService.UpdateArticle(c.Context(), message.ArticleID, updateReq)
		if err != nil {
			return response.Error(c, err)
		}

		// Mark artifact as applied
		if err := chatService.MarkArtifactAsApplied(c.Context(), messageID); err != nil {
			return response.Error(c, err)
		}

		return response.Success(c, fiber.Map{
			"status":     "accepted",
			"message_id": messageID,
		})
	}
}

// RejectArtifactHandler handles rejecting an artifact
func RejectArtifactHandler(chatService *chat.MessageService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		messageIDStr := c.Params("messageId")
		messageID, err := uuid.Parse(messageIDStr)
		if err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid message ID"))
		}

		var req struct {
			Reason string `json:"reason"`
		}
		if err := c.BodyParser(&req); err != nil {
			// Reason is optional
			req.Reason = ""
		}

		// Reject the artifact
		if err := chatService.RejectArtifact(c.Context(), messageID, req.Reason); err != nil {
			return response.Error(c, err)
		}

		return response.Success(c, fiber.Map{
			"status":     "rejected",
			"message_id": messageID,
		})
	}
}

// GetConversationHistoryHandler retrieves chat messages for an article
func GetConversationHistoryHandler(chatService *chat.MessageService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		articleIDStr := c.Params("articleId")
		articleID, err := uuid.Parse(articleIDStr)
		if err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid article ID"))
		}

		limit := c.QueryInt("limit", 50)

		messages, err := chatService.GetConversationHistory(c.Context(), articleID, limit)
		if err != nil {
			return response.Error(c, err)
		}

		return response.Success(c, fiber.Map{
			"messages":   messages,
			"article_id": articleID,
			"total":      len(messages),
		})
	}
}

// GetPendingArtifactsHandler retrieves pending artifacts for an article
func GetPendingArtifactsHandler(chatService *chat.MessageService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		articleIDStr := c.Params("articleId")
		articleID, err := uuid.Parse(articleIDStr)
		if err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid article ID"))
		}

		messages, err := chatService.GetPendingArtifacts(c.Context(), articleID)
		if err != nil {
			return response.Error(c, err)
		}

		return response.Success(c, fiber.Map{
			"artifacts":  messages,
			"article_id": articleID,
			"total":      len(messages),
		})
	}
}

