package main

import (
	"blog-agent-go/backend/internal/database"
	"blog-agent-go/backend/internal/server"
	"blog-agent-go/backend/internal/services"
	"fmt"
	"log"
	"os"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	// Load environment variables
	secretKey := os.Getenv("AUTH_SECRET")
	if secretKey == "" {
		log.Fatal("AUTH_SECRET environment variable is required")
	}

	// Initialize database service
	dbService := database.New()

	// Initialize AWS S3 client
	s3Client := services.NewR2S3Client()
	bucket := os.Getenv("S3_BUCKET")
	if bucket == "" {
		log.Fatal("S3_BUCKET environment variable is required")
	}

	urlPrefix := os.Getenv("S3_URL_PREFIX")
	if urlPrefix == "" {
		log.Fatal("S3_URL_PREFIX environment variable is required")
	}

	// Initialize services
	authService := services.NewAuthService(dbService, secretKey)
	writerAgent := services.NewWriterAgent()
	blogService := services.NewArticleService(dbService, writerAgent)
    projectsService := services.NewProjectsService(dbService)
	storageService := services.NewStorageService(s3Client, bucket, urlPrefix)
	imageService := services.NewImageGenerationService(dbService, storageService)
	pagesService := services.NewPagesService(dbService)

	// Initialize the Agent-powered copilot manager
	if err := services.InitializeAgentCopilotManager(); err != nil {
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
		services.GetAgentAsyncCopilotManager(),
	)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	address := fmt.Sprintf(":%s", port)
	log.Printf("Attempting to start server on address: %s", address)
	if err := srv.App.Listen(address); err != nil {
		log.Fatalf("Failed to bind to %s: %v", address, err)
	}
}
