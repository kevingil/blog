package database

import (
	"fmt"
	"log"
	"os"

	"blog-agent-go/backend/models"

	libsql "github.com/ekristen/gorm-libsql"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
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
	GetUserByEmail(email string) (*models.User, error)
	CreateUser(user *models.User) error
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

	dburl := os.Getenv("DB_URL")

	// Require DB_URL for libsql connection
	if dburl == "" {
		log.Fatal("DB_URL environment variable is required for libsql connection")
	}

	// Configure GORM logger
	gormLogger := logger.Default.LogMode(logger.Info)

	// Connect using libsql driver
	db, err := gorm.Open(libsql.Open(dburl), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		log.Fatalf("Failed to connect to libsql database '%s': %v", dburl, err)
	}

	// Get underlying sql.DB for health checks and connection pool configuration
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get underlying sql.DB:", err)
	}

	// Configure connection pool (conservative settings for Turso)
	sqlDB.SetMaxOpenConns(5)
	sqlDB.SetMaxIdleConns(5)

	// Auto-migrate all models (commented out - models now match existing schema)
	// log.Println("Running database migrations...")
	// err = db.AutoMigrate(
	// 	&models.User{},
	// 	&models.Article{},
	// 	&models.Tag{},
	// 	&models.ArticleTag{},
	// 	&models.ImageGeneration{},
	// 	&models.AboutPage{},
	// 	&models.ContactPage{},
	// 	&models.Project{},
	// )
	// if err != nil {
	// 	log.Fatal("Failed to auto-migrate models:", err)
	// }

	log.Printf("Successfully connected to libsql database")
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

func (s *service) GetUserByEmail(email string) (*models.User, error) {
	var user models.User

	result := s.db.Where("email = ?", email).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}

	return &user, nil
}

func (s *service) CreateUser(user *models.User) error {
	result := s.db.Create(user)
	return result.Error
}

func (s *service) GetDB() *gorm.DB {
	return s.db
}
