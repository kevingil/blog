package main

import (
	"blog-agent-go/backend/database"
	"blog-agent-go/backend/server"
	"blog-agent-go/backend/services"
	"log"
	"os"

	_ "github.com/joho/godotenv/autoload"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// Load environment variables
	secretKey := os.Getenv("AUTH_SECRET")
	if secretKey == "" {
		log.Fatal("AUTH_SECRET environment variable is required")
	}

	// Initialize database
	dbService := database.New()

	// Initialize GORM with the database connection
	db, err := gorm.Open(sqlite.New(sqlite.Config{
		Conn: dbService.GetDB(),
	}), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to initialize GORM: %v", err)
	}

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
	authService := services.NewAuthService(db, secretKey)
	writerAgent := services.NewWriterAgent(os.Getenv("ANTHROPIC_API_KEY"))
	blogService := services.NewArticleService(db, writerAgent)
	imageService := services.NewImageGenerationService(db)
	storageService := services.NewStorageService(s3Client, bucket, urlPrefix)
	pagesService := services.NewPagesService(db)

	// Initialize and start server
	srv := server.NewFiberServer(
		db,
		authService,
		blogService,
		imageService,
		storageService,
		pagesService,
	)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("listening on %s", port)
	if err := srv.App.Listen(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
