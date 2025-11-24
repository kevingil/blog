package controller

import (
	"blog-agent-go/backend/internal/core/chat"
	"blog-agent-go/backend/internal/errors"
	"blog-agent-go/backend/internal/response"
	"blog-agent-go/backend/internal/services"
	"blog-agent-go/backend/internal/websocket"
	"context"
	"encoding/json"
	"log"
	"strconv"

	websocketLib "github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

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

// GetConversationHistoryHandler returns the conversation history for an article
func GetConversationHistoryHandler(chatService *chat.MessageService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		articleID := c.Params("articleId")
		if articleID == "" {
			return response.Error(c, errors.NewInvalidInputError("Article ID is required"))
		}

		log.Printf("[ConversationHistory] Fetching messages for article: %s", articleID)

		articleUUID, err := uuid.Parse(articleID)
		if err != nil {
			log.Printf("[ConversationHistory] Invalid article ID format: %s", articleID)
			return response.Error(c, errors.NewInvalidInputError("Invalid article ID format"))
		}

		// Get limit from query params, default to 50
		limit := 50
		if limitStr := c.Query("limit"); limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
				limit = parsedLimit
			}
		}

		// Get conversation history
		messages, err := chatService.GetConversationHistory(c.Context(), articleUUID, limit)
		if err != nil {
			log.Printf("[ConversationHistory] ❌ Failed to fetch messages: %v", err)
			return response.Error(c, err)
		}

		log.Printf("[ConversationHistory] ✅ Found %d messages for article %s", len(messages), articleID)

		// Log each message for debugging
		for i, msg := range messages {
			log.Printf("[ConversationHistory]    [%d] %s: %s (ID: %s)",
				i+1,
				msg.Role,
				truncateString(msg.Content, 60),
				msg.ID,
			)

			// Log if message has metadata
			if len(msg.MetaData) > 2 && string(msg.MetaData) != "{}" {
				log.Printf("[ConversationHistory]        Has metadata: %d bytes", len(msg.MetaData))
			}
		}

		return response.Success(c, fiber.Map{
			"messages":   messages,
			"article_id": articleID,
			"total":      len(messages),
		})
	}
}
