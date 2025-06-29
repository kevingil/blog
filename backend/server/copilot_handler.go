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

// CopilotKitHandler handles POST /api/copilotkit
//
// The current implementation makes a single non-streaming request to the
// OpenAI API (to keep things simple) and then forwards the response to the
// frontend as an SSE stream. This preserves the streaming contract expected by
// the CopilotKit React runtime while significantly reducing backend
// complexity.
func (s *FiberServer) CopilotKitHandler(c *fiber.Ctx) error {
	var req services.ChatRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	// Create a tiny context with timeout so requests do not hang forever.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	copilot := services.NewCopilotKitService()
	assistantReply, err := copilot.Generate(ctx, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		// The simplest possible framing: one data chunk followed by the
		// mandatory [DONE] marker used by the OpenAI edge-compat layer.
		payload, _ := json.Marshal(fiber.Map{
			"role":    "assistant",
			"content": assistantReply,
		})
		fmt.Fprintf(w, "data: %s\n\n", payload)
		fmt.Fprint(w, "data: [DONE]\n\n")
		w.Flush()
	})

	return nil
}
