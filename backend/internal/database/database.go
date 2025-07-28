package database

import (
	"fmt"
	"log"
	"os"

	"blog-agent-go/backend/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Service represents a service that interacts with a database.
type Service interface {
	// Health returns a map of health status information.
	Health() map[string]string

	// Close terminates the database connection.
	Close() error

	// GetDB returns the underlying GORM database connection
	GetDB() *gorm.DB

	// User operations
	GetAccountByEmail(email string) (*models.Account, error)
	CreateAccount(account *models.Account) error
}

type service struct {
	db *gorm.DB
}

var (
	dbInstance *service
)

func New() Service {
	// Reuse Connection
	if dbInstance != nil {
		return dbInstance
	}

	dburl := os.Getenv("DATABASE_URL")

	if dburl == "" {
		log.Fatal("DATABASE_URL environment variable is required for PostgreSQL connection")
	}

	// Configure GORM logger
	gormLogger := logger.Default.LogMode(logger.Info)

	// Connect using PostgreSQL driver
	db, err := gorm.Open(postgres.Open(dburl), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL database '%s': %v", dburl, err)
	}

	// Get underlying sql.DB for health checks and connection pool configuration
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get underlying sql.DB:", err)
	}

	// Configure connection pool (adjust as needed for Postgres)
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(10)

	log.Printf("Successfully connected to PostgreSQL database")
	dbInstance = &service{
		db: db,
	}
	return dbInstance
}

// Health checks the health of the database connection by pinging the database.
func (s *service) Health() map[string]string {
	stats := make(map[string]string)

	// Get underlying sql.DB for health check
	sqlDB, err := s.db.DB()
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("failed to get sql.DB: %v", err)
		return stats
	}

	// Ping the database
	err = sqlDB.Ping()
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %v", err)
		return stats
	}

	// Database is up
	stats["status"] = "up"
	stats["message"] = "It's healthy"

	// Get database stats
	dbStats := sqlDB.Stats()
	stats["open_connections"] = fmt.Sprintf("%d", dbStats.OpenConnections)
	stats["in_use"] = fmt.Sprintf("%d", dbStats.InUse)
	stats["idle"] = fmt.Sprintf("%d", dbStats.Idle)

	return stats
}

// Close closes the database connection.
func (s *service) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	log.Println("Closing database connection")
	return sqlDB.Close()
}

func (s *service) GetAccountByEmail(email string) (*models.Account, error) {
	var account models.Account
	result := s.db.Where("email = ?", email).First(&account)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &account, nil
}

func (s *service) CreateAccount(account *models.Account) error {
	result := s.db.Create(account)
	return result.Error
}

func (s *service) GetDB() *gorm.DB {
	return s.db
}
