// Package agent provides types and infrastructure for the agent system
package agent

import (
	"os"
	"strconv"
	"time"
)

// Config holds configuration for the agent system
type Config struct {
	MaxConcurrentRequests int
	RequestTimeout        time.Duration
	ChannelBuffer         int
	CleanupDelay          time.Duration
	DefaultModel          string
}

// LoadConfig loads agent configuration from environment variables
func LoadConfig() Config {
	return Config{
		MaxConcurrentRequests: getEnvInt("AGENT_MAX_CONCURRENT", 10),
		RequestTimeout:        getEnvDuration("AGENT_REQUEST_TIMEOUT", 10*time.Minute),
		ChannelBuffer:         getEnvInt("AGENT_CHANNEL_BUFFER", 100),
		CleanupDelay:          getEnvDuration("AGENT_CLEANUP_DELAY", 15*time.Minute),
		DefaultModel:          getEnvString("AGENT_DEFAULT_MODEL", "gpt-4o"),
	}
}

// getEnvInt gets an integer from environment or returns default
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvDuration gets a duration from environment (in minutes) or returns default
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return time.Duration(intValue) * time.Minute
		}
	}
	return defaultValue
}

// getEnvString gets a string from environment or returns default
func getEnvString(key string, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

