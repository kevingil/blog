package server

import (
	"log"

	"blog-agent-go/backend/services"

	"github.com/gofiber/fiber/v2"
)

// WritingCopilotHandler handles POST agent/writing_copilot
//
// This implementation now uses async background processing with WebSocket streaming.
// The handler returns immediately with a request ID, and the actual processing
// happens in the background with results streamed via WebSocket.
func (s *FiberServer) WritingCopilotHandler(c *fiber.Ctx) error {
	var req services.ChatRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("WritingCopilotHandler: Failed to parse request body: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	log.Printf("WritingCopilotHandler: Received request with %d messages", len(req.Messages))
	if req.DocumentContent != "" {
		log.Printf("WritingCopilotHandler: Document content length: %d", len(req.DocumentContent))
	}

	// Get the async manager and submit the request
	manager := services.GetAsyncCopilotManager()
	requestID, err := manager.SubmitChatRequest(req)
	if err != nil {
		log.Printf("WritingCopilotHandler: Failed to submit request: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	log.Printf("WritingCopilotHandler: Submitted async request with ID: %s", requestID)

	// Return immediately with the request ID
	return c.JSON(services.ChatRequestResponse{
		RequestID: requestID,
		Status:    "processing",
	})
}
