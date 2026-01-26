// Package agent provides HTTP handlers for AI agent and websocket functionality
package agent

import (
	"backend/pkg/api/middleware"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

// Register registers agent routes on the app
func Register(app *fiber.App) {
	// Copilot - Agent-powered writing assistant (requires authentication)
	app.Post("/agent", middleware.Auth(), AgentCopilot)
	app.Get("/websocket", websocket.New(WebsocketHandler))

	// Agent - Conversation and artifact management (requires authentication)
	agentGroup := app.Group("/agent", middleware.Auth())
	agentGroup.Get("/conversations/:articleId", GetConversationHistory)
	agentGroup.Get("/artifacts/:articleId/pending", GetPendingArtifacts)
	agentGroup.Post("/artifacts/:messageId/accept", AcceptArtifact)
	agentGroup.Post("/artifacts/:messageId/reject", RejectArtifact)
}
