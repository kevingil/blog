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
	"backend/pkg/core/worker"
	"backend/pkg/database"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

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

	// Initialize Worker Manager
	workerLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	workerManager := worker.NewWorkerManager(workerLogger)

	// Create and register workers
	crawlWorker := worker.NewCrawlWorker(workerLogger, cfg.Worker.ExaAPIKey)
	insightWorker := worker.NewInsightWorker(workerLogger, cfg.Worker.GroqAPIKey)
	discoveryWorker := worker.NewDiscoveryWorker(workerLogger, cfg.Worker.ExaAPIKey)

	workerManager.RegisterWorker(crawlWorker)
	workerManager.RegisterWorker(insightWorker)
	workerManager.RegisterWorker(discoveryWorker)

	// NOTE: Cron scheduling is disabled for now - workers run manually only
	// To enable scheduled runs, uncomment the following and set environment variables:
	// WORKER_CRAWL_SCHEDULE, WORKER_INSIGHT_SCHEDULE, WORKER_DISCOVERY_SCHEDULE
	// Schedule format: "seconds minutes hours day-of-month month day-of-week"
	// Examples: "0 */15 * * * *" (every 15 mins), "0 0 */6 * * *" (every 6 hours)
	//
	// if cfg.Worker.CrawlSchedule != "" {
	// 	if err := workerManager.ScheduleWorker("crawl", cfg.Worker.CrawlSchedule); err != nil {
	// 		log.Printf("Warning: Failed to schedule crawl worker: %v", err)
	// 	}
	// }
	// if cfg.Worker.InsightSchedule != "" {
	// 	if err := workerManager.ScheduleWorker("insight", cfg.Worker.InsightSchedule); err != nil {
	// 		log.Printf("Warning: Failed to schedule insight worker: %v", err)
	// 	}
	// }
	// if cfg.Worker.DiscoverySchedule != "" {
	// 	if err := workerManager.ScheduleWorker("discovery", cfg.Worker.DiscoverySchedule); err != nil {
	// 		log.Printf("Warning: Failed to schedule discovery worker: %v", err)
	// 	}
	// }

	// Start worker manager
	workerManager.Start()
	worker.SetGlobalManager(workerManager)
	log.Printf("Initialized Worker Manager with %d workers", len(workerManager.GetRegisteredWorkers()))

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

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down...")
		workerManager.Stop()
		if err := app.Shutdown(); err != nil {
			log.Printf("Error shutting down server: %v", err)
		}
	}()

	// Start server
	address := fmt.Sprintf(":%s", cfg.Server.Port)
	log.Printf("Starting server on %s", address)
	if err := app.Listen(address); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
