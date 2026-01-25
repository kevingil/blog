package agent

import (
	"backend/pkg/api/response"
	"backend/pkg/api/services"
	agentws "backend/pkg/api/websocket"
	"backend/pkg/core"
	"backend/pkg/core/chat"
	"backend/pkg/database"
	"context"
	"encoding/json"
	"log"
	"strconv"

	websocketLib "github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// chatSvc is a lazy-initialized chat service
var chatSvc *chat.MessageService

func getChatService() *chat.MessageService {
	if chatSvc == nil {
		chatSvc = chat.NewMessageService(database.New())
	}
	return chatSvc
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// AgentCopilot handles POST /agent
func AgentCopilot(c *fiber.Ctx) error {
	var req services.ChatRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}

	manager := services.GetAgentAsyncCopilotManager()
	requestID, err := manager.SubmitChatRequest(req)
	if err != nil {
		log.Printf("[Agent API] Failed to submit request: %v", err)
		return response.Error(c, err)
	}

	return response.Success(c, services.ChatRequestResponse{
		RequestID: requestID,
		Status:    "processing",
	})
}

// WebsocketHandler handles GET /websocket
func WebsocketHandler(con *websocketLib.Conn) {
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
					agentws.HandleAgentStream(ctx, con, msg.RequestID, agentManager)
				}
			}
		}
	}()
	<-ctx.Done()
}

// GetConversationHistory handles GET /agent/conversations/:articleId
func GetConversationHistory(c *fiber.Ctx) error {
	articleID := c.Params("articleId")
	if articleID == "" {
		return response.Error(c, core.InvalidInputError("Article ID is required"))
	}

	log.Printf("[ConversationHistory] Fetching messages for article: %s", articleID)

	articleUUID, err := uuid.Parse(articleID)
	if err != nil {
		log.Printf("[ConversationHistory] Invalid article ID format: %s", articleID)
		return response.Error(c, core.InvalidInputError("Invalid article ID format"))
	}

	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	messages, err := getChatService().GetConversationHistory(c.Context(), articleUUID, limit)
	if err != nil {
		log.Printf("[ConversationHistory] Failed to fetch messages: %v", err)
		return response.Error(c, err)
	}

	log.Printf("[ConversationHistory] Found %d messages for article %s", len(messages), articleID)

	return response.Success(c, fiber.Map{
		"messages":   messages,
		"article_id": articleID,
		"total":      len(messages),
	})
}

// GetPendingArtifacts handles GET /agent/artifacts/:articleId/pending
func GetPendingArtifacts(c *fiber.Ctx) error {
	articleIDStr := c.Params("articleId")
	articleID, err := uuid.Parse(articleIDStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid article ID"))
	}

	artifacts, err := getChatService().GetPendingArtifacts(c.Context(), articleID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{
		"artifacts": artifacts,
	})
}

// AcceptArtifact handles POST /agent/artifacts/:messageId/accept
func AcceptArtifact(c *fiber.Ctx) error {
	messageIDStr := c.Params("messageId")
	messageID, err := uuid.Parse(messageIDStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid message ID"))
	}

	var req struct {
		Feedback string `json:"feedback"`
	}
	_ = c.BodyParser(&req) // Feedback is optional

	if err := getChatService().AcceptArtifact(c.Context(), messageID, req.Feedback); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{"success": true})
}

// RejectArtifact handles POST /agent/artifacts/:messageId/reject
func RejectArtifact(c *fiber.Ctx) error {
	messageIDStr := c.Params("messageId")
	messageID, err := uuid.Parse(messageIDStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid message ID"))
	}

	var req struct {
		Feedback string `json:"feedback"`
	}
	_ = c.BodyParser(&req) // Feedback is optional

	if err := getChatService().RejectArtifact(c.Context(), messageID, req.Feedback); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{"success": true})
}
