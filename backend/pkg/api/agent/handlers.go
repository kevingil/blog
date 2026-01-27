package agent

import (
	"backend/pkg/api/response"
	agentws "backend/pkg/api/websocket"
	"backend/pkg/core"
	coreAgent "backend/pkg/core/agent"
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

// AgentCopilot handles POST /agent
// @Summary Submit agent request
// @Description Submit a chat request to the AI agent for processing
// @Tags agent
// @Accept json
// @Produce json
// @Param request body object true "Chat request"
// @Success 200 {object} response.SuccessResponse{data=object{request_id=string,status=string}}
// @Failure 400 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /agent [post]
func AgentCopilot(c *fiber.Ctx) error {
	var req coreAgent.ChatRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}

	manager := coreAgent.GetAgentAsyncCopilotManager()
	requestID, err := manager.SubmitChatRequest(req)
	if err != nil {
		log.Printf("[Agent API] Failed to submit request: %v", err)
		return response.Error(c, err)
	}

	return response.Success(c, coreAgent.ChatRequestResponse{
		RequestID: requestID,
		Status:    "processing",
	})
}

// WebsocketHandler handles GET /websocket
func WebsocketHandler(con *websocketLib.Conn) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	agentManager := coreAgent.GetAgentAsyncCopilotManager()

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
// @Summary Get conversation history
// @Description Get the chat history for an article
// @Tags agent
// @Accept json
// @Produce json
// @Param articleId path string true "Article ID"
// @Param limit query int false "Max messages to return" default(50)
// @Success 200 {object} response.SuccessResponse{data=object{messages=[]object,article_id=string,total=int}}
// @Failure 400 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /agent/conversations/{articleId} [get]
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

// ClearConversationHistory handles DELETE /agent/conversations/:articleId
// @Summary Clear conversation history
// @Description Clear all chat messages for an article
// @Tags agent
// @Accept json
// @Produce json
// @Param articleId path string true "Article ID"
// @Success 200 {object} response.SuccessResponse{data=object{success=boolean}}
// @Failure 400 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /agent/conversations/{articleId} [delete]
func ClearConversationHistory(c *fiber.Ctx) error {
	articleID := c.Params("articleId")
	if articleID == "" {
		return response.Error(c, core.InvalidInputError("Article ID is required"))
	}

	log.Printf("[ConversationHistory] Clearing messages for article: %s", articleID)

	articleUUID, err := uuid.Parse(articleID)
	if err != nil {
		log.Printf("[ConversationHistory] Invalid article ID format: %s", articleID)
		return response.Error(c, core.InvalidInputError("Invalid article ID format"))
	}

	if err := getChatService().ClearConversationHistory(c.Context(), articleUUID); err != nil {
		log.Printf("[ConversationHistory] Failed to clear messages: %v", err)
		return response.Error(c, err)
	}

	log.Printf("[ConversationHistory] Successfully cleared messages for article %s", articleID)

	return response.Success(c, fiber.Map{"success": true})
}

// GetPendingArtifacts handles GET /agent/artifacts/:articleId/pending
// @Summary Get pending artifacts
// @Description Get all pending artifacts for an article
// @Tags agent
// @Accept json
// @Produce json
// @Param articleId path string true "Article ID"
// @Success 200 {object} response.SuccessResponse{data=object{artifacts=[]object}}
// @Failure 400 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /agent/artifacts/{articleId}/pending [get]
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
// @Summary Accept artifact
// @Description Accept a pending artifact
// @Tags agent
// @Accept json
// @Produce json
// @Param messageId path string true "Message ID"
// @Param request body object{feedback=string} false "Optional feedback"
// @Success 200 {object} response.SuccessResponse{data=object{success=boolean}}
// @Failure 400 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /agent/artifacts/{messageId}/accept [post]
func AcceptArtifact(c *fiber.Ctx) error {
	messageIDStr := c.Params("messageId")
	messageID, err := uuid.Parse(messageIDStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid message ID"))
	}

	var req struct {
		Feedback string `json:"feedback"`
	}
	_ = c.BodyParser(&req)

	if err := getChatService().AcceptArtifact(c.Context(), messageID, req.Feedback); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{"success": true})
}

// RejectArtifact handles POST /agent/artifacts/:messageId/reject
// @Summary Reject artifact
// @Description Reject a pending artifact
// @Tags agent
// @Accept json
// @Produce json
// @Param messageId path string true "Message ID"
// @Param request body object{feedback=string} false "Optional feedback"
// @Success 200 {object} response.SuccessResponse{data=object{success=boolean}}
// @Failure 400 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /agent/artifacts/{messageId}/reject [post]
func RejectArtifact(c *fiber.Ctx) error {
	messageIDStr := c.Params("messageId")
	messageID, err := uuid.Parse(messageIDStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid message ID"))
	}

	var req struct {
		Feedback string `json:"feedback"`
	}
	_ = c.BodyParser(&req)

	if err := getChatService().RejectArtifact(c.Context(), messageID, req.Feedback); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{"success": true})
}
