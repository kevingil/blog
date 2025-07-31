package tracing

import (
	"context"
	"fmt"
	"time"

	"blog-agent-go/backend/internal/llm/config"
	"blog-agent-go/backend/internal/llm/logging"
)

// InitializeFromConfig initializes tracing from the provided configuration
func InitializeFromConfig(tracingConfig *config.TracingConfig) error {
	if err := Initialize(tracingConfig); err != nil {
		return fmt.Errorf("failed to initialize tracing: %w", err)
	}

	if tracingConfig.Enabled {
		logging.Info("Tracing initialized",
			"endpoint", tracingConfig.Endpoint,
			"service", tracingConfig.ServiceName,
			"version", tracingConfig.ServiceVersion,
		)
	} else {
		logging.Info("Tracing disabled")
	}

	return nil
}

// GracefulShutdown shuts down tracing with a timeout
func GracefulShutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := Shutdown(ctx); err != nil {
		logging.ErrorPersist(fmt.Sprintf("Failed to shutdown tracing gracefully: %v", err))
		return err
	}

	logging.Info("Tracing shutdown completed")
	return nil
}
