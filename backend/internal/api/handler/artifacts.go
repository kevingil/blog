package handler

import (
	"blog-agent-go/backend/internal/core/chat"
	"blog-agent-go/backend/internal/errors"
	"blog-agent-go/backend/internal/response"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// AcceptArtifactRequest represents the request body for accepting an artifact
type AcceptArtifactRequest struct {
	Feedback string `json:"feedback"`
}

// RejectArtifactRequest represents the request body for rejecting an artifact
type RejectArtifactRequest struct {
	Reason string `json:"reason"`
}

// AcceptArtifact marks an artifact as accepted
func AcceptArtifact() fiber.Handler {
	return func(c *fiber.Ctx) error {
		messageID := c.Params("messageId")
		if messageID == "" {
			return response.Error(c, errors.NewInvalidInputError("Message ID is required"))
		}

		messageUUID, err := uuid.Parse(messageID)
		if err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid message ID format"))
		}

		// Parse request body
		var req AcceptArtifactRequest
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid request body"))
		}

		// Get chat service from context (injected by dependency injection in routes)
		chatService := c.Locals("chatService").(*chat.MessageService)
		if chatService == nil {
			return response.Error(c, errors.NewInternalError("Chat service not available"))
		}

		// Accept the artifact
		if err := chatService.AcceptArtifact(c.Context(), messageUUID, req.Feedback); err != nil {
			return response.Error(c, err)
		}

		return response.Success(c, fiber.Map{
			"status":     "accepted",
			"message_id": messageID,
		})
	}
}

// RejectArtifact marks an artifact as rejected
func RejectArtifact() fiber.Handler {
	return func(c *fiber.Ctx) error {
		messageID := c.Params("messageId")
		if messageID == "" {
			return response.Error(c, errors.NewInvalidInputError("Message ID is required"))
		}

		messageUUID, err := uuid.Parse(messageID)
		if err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid message ID format"))
		}

		// Parse request body
		var req RejectArtifactRequest
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid request body"))
		}

		// Get chat service from context
		chatService := c.Locals("chatService").(*chat.MessageService)
		if chatService == nil {
			return response.Error(c, errors.NewInternalError("Chat service not available"))
		}

		// Reject the artifact
		if err := chatService.RejectArtifact(c.Context(), messageUUID, req.Reason); err != nil {
			return response.Error(c, err)
		}

		return response.Success(c, fiber.Map{
			"status":     "rejected",
			"message_id": messageID,
		})
	}
}

// GetPendingArtifacts returns all pending artifacts for an article
func GetPendingArtifacts() fiber.Handler {
	return func(c *fiber.Ctx) error {
		articleID := c.Params("articleId")
		if articleID == "" {
			return response.Error(c, errors.NewInvalidInputError("Article ID is required"))
		}

		articleUUID, err := uuid.Parse(articleID)
		if err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid article ID format"))
		}

		// Get chat service from context
		chatService := c.Locals("chatService").(*chat.MessageService)
		if chatService == nil {
			return response.Error(c, errors.NewInternalError("Chat service not available"))
		}

		// Get pending artifacts
		messages, err := chatService.GetPendingArtifacts(c.Context(), articleUUID)
		if err != nil {
			return response.Error(c, err)
		}

		return response.Success(c, fiber.Map{
			"artifacts": messages,
			"total":     len(messages),
		})
	}
}
