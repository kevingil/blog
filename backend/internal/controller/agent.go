package controller

import (
	"blog-agent-go/backend/internal/errors"
	"blog-agent-go/backend/internal/response"
	"blog-agent-go/backend/internal/services"
	"blog-agent-go/backend/internal/websocket"
	"context"
	"encoding/json"
	"log"

	websocketLib "github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

// AgentCopilotHandler - Agent-powered writing assistant
func AgentCopilotHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req services.ChatRequest
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid request body"))
		}

		// Get the agent async manager and submit the request
		manager := services.GetAgentAsyncCopilotManager()
		requestID, err := manager.SubmitChatRequest(req)
		if err != nil {
			log.Printf("[Agent API] Failed to submit request: %v", err)
			return response.Error(c, err)
		}

		// Return immediately with the request ID
		return response.Success(c, services.ChatRequestResponse{
			RequestID: requestID,
			Status:    "processing",
		})
	}
}

func WebsocketHandler() func(*websocketLib.Conn) {
	return func(con *websocketLib.Conn) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		agentManager := services.GetAgentAsyncCopilotManager()

		go func() {
			defer cancel()
			for {
				messageType, message, err := con.ReadMessage()
				if err != nil {
					break
				}
				if messageType == websocketLib.TextMessage {
					var msg struct {
						RequestID string `json:"requestId"`
						Action    string `json:"action"`
					}
					if err := json.Unmarshal(message, &msg); err != nil {
						continue
					}
					if msg.Action == "subscribe" && msg.RequestID != "" {
						websocket.HandleAgentStream(ctx, con, msg.RequestID, agentManager)
					}
				}
			}
		}()
		<-ctx.Done()
	}
}
