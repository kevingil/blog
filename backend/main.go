package main

import (
	"blog-agent-go/backend/internal/config"
	"blog-agent-go/backend/internal/core/chat"
	"blog-agent-go/backend/internal/database"
	"blog-agent-go/backend/internal/server"
	"blog-agent-go/backend/internal/services"
	"fmt"
	"log"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	// Load and validate configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database service
	dbService := database.New()

	// Initialize AWS S3 client
	s3Client := services.NewR2S3Client()

	// Initialize services
	authService := services.NewAuthService(dbService, cfg.Auth.SecretKey)
	writerAgent := services.NewWriterAgent()
	blogService := services.NewArticleService(dbService, writerAgent)
	projectsService := services.NewProjectsService(dbService)
	storageService := services.NewStorageService(s3Client, cfg.AWS.S3Bucket, cfg.AWS.S3URLPrefix)
	imageService := services.NewImageGenerationService(dbService, storageService)
	pagesService := services.NewPagesService(dbService)
	sourcesService := services.NewArticleSourceService(dbService)
	chatService := chat.NewMessageService(dbService)
	profileService := services.NewProfileService(dbService)
	organizationService := services.NewOrganizationService(dbService)

	// Initialize the Agent-powered copilot manager with sources service and chat service
	if err := services.InitializeAgentCopilotManager(sourcesService, chatService); err != nil {
		log.Printf("Warning: Failed to initialize AgentCopilotManager: %v", err)
	}

	log.Printf("Initialized Agent Services; AsyncCopilotManager and AgentCopilotManager")

	// Initialize and start server
	srv := server.NewFiberServer(
		dbService,
		authService,
		blogService,
		projectsService,
		imageService,
		storageService,
		pagesService,
		sourcesService,
		chatService,
		services.GetAgentAsyncCopilotManager(),
		profileService,
		organizationService,
	)

	address := fmt.Sprintf(":%s", cfg.Server.Port)
	log.Printf("Attempting to start server on address: %s", address)
	if err := srv.App.Listen(address); err != nil {
		log.Fatalf("Failed to bind to %s: %v", address, err)
	}
}
