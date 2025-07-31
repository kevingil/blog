package main

import (
	"blog-agent-go/backend/internal/database"
	"blog-agent-go/backend/internal/llm/config"
	"blog-agent-go/backend/internal/llm/tracing"
	"blog-agent-go/backend/internal/server"
	"blog-agent-go/backend/internal/services"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	// Load environment variables
	secretKey := os.Getenv("AUTH_SECRET")
	if secretKey == "" {
		log.Fatal("AUTH_SECRET environment variable is required")
	}

	// Initialize tracing service
	cfg := config.Get()
	if err := tracing.InitializeFromConfig(cfg.Tracing); err != nil {
		log.Printf("Warning: Failed to initialize tracing: %v", err)
	}

	// Set up graceful shutdown for tracing
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down tracing...")
		if err := tracing.GracefulShutdown(); err != nil {
			log.Printf("Error during tracing shutdown: %v", err)
		}
		os.Exit(0)
	}()

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
