// Package copilot provides the interactive article copilot orchestration layer.
package copilot

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
}

// LoadConfig loads agent configuration from environment variables
func LoadConfig() Config {
	return Config{
		MaxConcurrentRequests: getEnvInt("AGENT_MAX_CONCURRENT", 10),
		RequestTimeout:        getEnvDuration("AGENT_REQUEST_TIMEOUT", 10*time.Minute),
		ChannelBuffer:         getEnvInt("AGENT_CHANNEL_BUFFER", 100),
		CleanupDelay:          getEnvDuration("AGENT_CLEANUP_DELAY", 15*time.Minute),
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
