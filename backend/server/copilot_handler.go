package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"blog-agent-go/backend/services"

	"github.com/gofiber/fiber/v2"
)

// CopilotKitHandler handles POST agent/writing_copilot
//
// This implementation uses streaming responses from the OpenAI API and forwards
// them to the frontend as Server-Sent Events (SSE). This provides real-time
// streaming while maintaining compatibility with the expected frontend interface.
func (s *FiberServer) CopilotKitHandler(c *fiber.Ctx) error {
	var req services.ChatRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	// Create a context with timeout so requests do not hang forever.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	copilot := services.NewCopilotKitService()
	streamChan, err := copilot.GenerateStream(ctx, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		for response := range streamChan {
			if response.Done {
				fmt.Fprint(w, "data: [DONE]\n\n")
				break
			}

			payload, _ := json.Marshal(response)
			fmt.Fprintf(w, "data: %s\n\n", payload)
			w.Flush()
		}
	})

	return nil
}
