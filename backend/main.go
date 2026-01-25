// Package main is the entry point for the blog-agent API server.
//
//	@title			Blog Agent API
//	@version		1.0
//	@description	A blog management API with AI-powered writing assistance
//	@termsOfService	http://swagger.io/terms/
//
//	@contact.name	API Support
//	@contact.email	support@example.com
//
//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html
//
//	@host		localhost:8080
//	@BasePath	/
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Enter token with Bearer prefix: Bearer <token>
package main

import (
	"backend/pkg/api"
	coreAgent "backend/pkg/core/agent"
	"backend/pkg/config"
	"backend/pkg/core/chat"
	"backend/pkg/database"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	// Initialize configuration (makes it available via config.Get())
	config.Init()
	cfg := config.Get()

	// Initialize database (makes it available via database.DB())
	database.Init()

	// Initialize Agent-powered copilot manager with chat service
	chatService := chat.NewMessageService(database.New())
	if err := coreAgent.InitializeWithDefaults(chatService); err != nil {
		log.Printf("Warning: Failed to initialize AgentCopilotManager: %v", err)
	}
	log.Printf("Initialized Agent Services")

	// Create Fiber app and register routes
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Register all API routes
	api.RegisterRoutes(app)

	// Start server
	address := fmt.Sprintf(":%s", cfg.Server.Port)
	log.Printf("Starting server on %s", address)
	if err := app.Listen(address); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
