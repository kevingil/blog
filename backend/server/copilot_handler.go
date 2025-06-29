package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"blog-agent-go/backend/services"

	"github.com/gofiber/fiber/v2"
)

// CopilotKitHandler handles POST agent/writing_copilot
//
// This implementation uses streaming responses from the OpenAI API and forwards
// them to the frontend as Server-Sent Events (SSE). This provides real-time
// streaming while maintaining compatibility with the expected frontend interface.
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

	// Create a context with timeout so requests do not hang forever.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	copilot := services.NewCopilotKitService()
	streamChan, err := copilot.GenerateStream(ctx, req)
	if err != nil {
		log.Printf("WritingCopilotHandler: Failed to generate stream: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	log.Printf("WritingCopilotHandler: Starting SSE stream")

	// SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		responseCount := 0
		for response := range streamChan {
			responseCount++

			if response.Done {
				log.Printf("WritingCopilotHandler: Stream completed after %d responses", responseCount)
				fmt.Fprint(w, "data: [DONE]\n\n")
				break
			}

			if response.Error != "" {
				log.Printf("WritingCopilotHandler: Stream error after %d responses: %s", responseCount, response.Error)
				// Send error as JSON and then finish
				payload, _ := json.Marshal(response)
				fmt.Fprintf(w, "data: %s\n\n", payload)
				w.Flush()
				fmt.Fprint(w, "data: [DONE]\n\n")
				break
			}

			payload, _ := json.Marshal(response)
			fmt.Fprintf(w, "data: %s\n\n", payload)
			w.Flush()
		}
		log.Printf("WritingCopilotHandler: SSE stream ended")
	})

	return nil
}
