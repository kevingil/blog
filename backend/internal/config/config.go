// Package config provides centralized configuration management for the application
package config

import (
	"fmt"
	"os"
)

// Config holds all configuration for the application loaded from environment variables.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
	AWS      AWSConfig
	CORS     CORSConfig
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Port string
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	URL string
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	SecretKey string
}

// AWSConfig holds AWS-related configuration
type AWSConfig struct {
	S3Bucket    string
	S3URLPrefix string
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins string
}

// Load loads configuration from environment variables and validates required fields.
// Returns an error if any required configuration is missing.
func Load() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Port: getEnvOrDefault("PORT", "8080"),
		},
		Database: DatabaseConfig{
			URL: os.Getenv("DATABASE_URL"),
		},
		Auth: AuthConfig{
			SecretKey: os.Getenv("AUTH_SECRET"),
		},
		AWS: AWSConfig{
			S3Bucket:    os.Getenv("S3_BUCKET"),
			S3URLPrefix: os.Getenv("S3_URL_PREFIX"),
		},
		CORS: CORSConfig{
			AllowedOrigins: getEnvOrDefault("ALLOWED_ORIGINS", ""),
		},
	}

	// Validate required fields
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate checks that all required configuration is present.
// Returns an error describing which required field is missing.
func (c *Config) Validate() error {
	if c.Database.URL == "" {
		return fmt.Errorf("DATABASE_URL environment variable is required")
	}
	if c.Auth.SecretKey == "" {
		return fmt.Errorf("AUTH_SECRET environment variable is required")
	}
	if c.AWS.S3Bucket == "" {
		return fmt.Errorf("S3_BUCKET environment variable is required")
	}
	if c.AWS.S3URLPrefix == "" {
		return fmt.Errorf("S3_URL_PREFIX environment variable is required")
	}
	return nil
}

// getEnvOrDefault gets an environment variable or returns a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

