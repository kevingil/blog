package controller

import (
	"blog-agent-go/backend/internal/services"
	"context"
	"encoding/json"
	"log"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

// AgentCopilotHandler - Agent-powered writing assistant
func AgentCopilotHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req services.ChatRequest
		if err := c.BodyParser(&req); err != nil {
			log.Printf("WritingCopilotHandler: Failed to parse request body: %v", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}

		log.Printf("WritingCopilotHandler: Received request with %d messages", len(req.Messages))
		if req.DocumentContent != "" {
			log.Printf("WritingCopilotHandler: Document content length: %d", len(req.DocumentContent))
		}

		// Get the agent async manager and submit the request
		manager := services.GetAgentAsyncCopilotManager()
		requestID, err := manager.SubmitChatRequest(req)
		if err != nil {
			log.Printf("WritingCopilotHandler: Failed to submit request: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		log.Printf("WritingCopilotHandler: Submitted async agent request with ID: %s", requestID)

		// Return immediately with the request ID
		return c.JSON(services.ChatRequestResponse{
			RequestID: requestID,
			Status:    "processing",
		})
	}
}

func WebsocketHandler() func(*websocket.Conn) {
	return func(con *websocket.Conn) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		log.Printf("WebSocket: New connection established")
		go func() {
			defer cancel()
			for {
				messageType, message, err := con.ReadMessage()
				if err != nil {
					log.Printf("WebSocket read error: %v", err)
					break
				}
				if messageType == websocket.TextMessage {
					var msg struct {
						RequestID string `json:"requestId"`
						Action    string `json:"action"`
					}
					if err := json.Unmarshal(message, &msg); err != nil {
						log.Printf("WebSocket message parse error: %v", err)
						continue
					}
					if msg.Action == "subscribe" && msg.RequestID != "" {
						log.Printf("WebSocket: Subscribing to request %s", msg.RequestID)
						handleAgentCopilotStreaming(ctx, con, msg.RequestID)
					}
				}
			}
		}()
		<-ctx.Done()
		log.Println("WebSocket connection closed")
	}
}

func handleAgentCopilotStreaming(ctx context.Context, con *websocket.Conn, requestID string) {
	agentManager := services.GetAgentAsyncCopilotManager()
	responseChan, exists := agentManager.GetResponseChannel(requestID)
	if !exists {
		log.Printf("WebSocket: Request ID %s not found", requestID)
		errorMsg := services.StreamResponse{
			RequestID: requestID,
			Type:      "error",
			Error:     "Request not found",
			Done:      true,
		}
		if msgBytes, err := json.Marshal(errorMsg); err == nil {
			con.WriteMessage(websocket.TextMessage, msgBytes)
		}
		return
	}
	log.Printf("WebSocket: Starting agent stream for request %s", requestID)
	for {
		select {
		case response, ok := <-responseChan:
			if !ok {
				log.Printf("WebSocket: Response channel closed for request %s", requestID)
				return
			}
			response.RequestID = requestID
			switch response.Type {
			case "plan":
				log.Printf("WebSocket: Sending plan for request %s", requestID)
			case "artifact":
				log.Printf("WebSocket: Sending artifact update for request %s", requestID)
			case "chat":
				log.Printf("WebSocket: Sending chat message for request %s", requestID)
			case "error":
				log.Printf("WebSocket: Sending error for request %s: %s", requestID, response.Error)
			case "done":
				log.Printf("WebSocket: Sending completion signal for request %s", requestID)
			}
			responseBytes, err := json.Marshal(response)
			if err != nil {
				log.Printf("WebSocket: Failed to marshal response: %v", err)
				continue
			}
			if err := con.WriteMessage(websocket.TextMessage, responseBytes); err != nil {
				log.Printf("WebSocket: Failed to write message: %v", err)
				return
			}
			if response.Done || response.Error != "" {
				log.Printf("WebSocket: Stream completed for request %s", requestID)
				return
			}
		case <-ctx.Done():
			log.Printf("WebSocket: Context cancelled for request %s", requestID)
			return
		}
	}
}
