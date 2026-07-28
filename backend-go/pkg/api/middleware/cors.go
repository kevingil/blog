package middleware

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// GetCORSConfig returns CORS configuration for the application.
// Supports localhost origins for development by default, with ALLOWED_ORIGINS env override for production.
func GetCORSConfig() cors.Config {
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	
	// Default to localhost origins for development
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:3000,http://localhost:5173,http://localhost:8080,http://127.0.0.1:3000,http://127.0.0.1:5173,http://127.0.0.1:8080"
	}

	return cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS,PATCH",
		AllowHeaders:     "Accept,Authorization,Content-Type",
		AllowCredentials: true,
		MaxAge:           300,
	}
}

// CORS returns the configured CORS middleware with appropriate origin handling.
func CORS() fiber.Handler {
	return cors.New(GetCORSConfig())
}

