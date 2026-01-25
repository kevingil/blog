package router

import (
	"backend/pkg/api/handler"
	"backend/pkg/api/middleware"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

// RegisterAgentRoutes registers AI agent and websocket routes
func RegisterAgentRoutes(app *fiber.App, deps RouteDeps) {
	// Copilot - Agent-powered writing assistant (requires authentication)
	app.Post("/agent", middleware.AuthMiddleware(deps.AuthService), handler.AgentCopilotHandler())
	app.Get("/websocket", websocket.New(handler.WebsocketHandler()))

	// Agent - Conversation and artifact management (requires authentication)
	agentGroup := app.Group("/agent", middleware.AuthMiddleware(deps.AuthService))
	agentGroup.Get("/conversations/:articleId", handler.GetConversationHistoryHandler(deps.ChatService))

	// Artifact endpoints with chat service injection
	agentGroup.Get("/artifacts/:articleId/pending", func(c *fiber.Ctx) error {
		c.Locals("chatService", deps.ChatService)
		return handler.GetPendingArtifacts()(c)
	})
	agentGroup.Post("/artifacts/:messageId/accept", func(c *fiber.Ctx) error {
		c.Locals("chatService", deps.ChatService)
		return handler.AcceptArtifact()(c)
	})
	agentGroup.Post("/artifacts/:messageId/reject", func(c *fiber.Ctx) error {
		c.Locals("chatService", deps.ChatService)
		return handler.RejectArtifact()(c)
	})
}
