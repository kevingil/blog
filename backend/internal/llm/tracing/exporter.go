package tracing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"blog-agent-go/backend/internal/llm/config"
	"blog-agent-go/backend/internal/llm/logging"
)

type Exporter struct {
	config   *config.TracingConfig
	client   *http.Client
	stopChan chan struct{}
	wg       sync.WaitGroup
}

func NewExporter(tracingConfig *config.TracingConfig) (*Exporter, error) {
	if err := validateTracingConfig(tracingConfig); err != nil {
		return nil, fmt.Errorf("invalid tracing config: %w", err)
	}

	exporter := &Exporter{
		config: tracingConfig,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		stopChan: make(chan struct{}),
	}

	return exporter, nil
}

func (e *Exporter) Stop(ctx context.Context) error {
	if !e.config.Enabled {
		return nil
	}

	close(e.stopChan)
	return nil
}

// StartCall sends a call start event to W&B Weave
func (e *Exporter) StartCall(callStart CallStart) error {
	if !e.config.Enabled {
		return nil
	}

	request := CallStartRequest{Start: callStart}
	return e.sendCallEvent("https://trace.wandb.ai/call/start", request)
}

// EndCall sends a call end event to W&B Weave
func (e *Exporter) EndCall(callEnd CallEnd) error {
	if !e.config.Enabled {
		return nil
	}

	request := CallEndRequest{End: callEnd}
	return e.sendCallEvent("https://trace.wandb.ai/call/end", request)
}

func (e *Exporter) sendCallEvent(endpoint string, request interface{}) error {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal call data: %w", err)
	}

	if e.config.Debug {
		logging.Debug("[TRACING] Sending call event", "endpoint", endpoint, "data", string(jsonData))
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers for W&B Weave API
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.config.APIKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body for debugging
	if e.config.Debug {
		respBody := make([]byte, 1024)
		n, _ := resp.Body.Read(respBody)
		logging.Debug("[TRACING] Response", "status", resp.StatusCode, "body", string(respBody[:n]))
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("W&B Weave API returned status: %d", resp.StatusCode)
	}

	return nil
}

// GetStats returns current exporter statistics
func (e *Exporter) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"enabled":      e.config.Enabled,
		"service_name": e.config.ServiceName,
		"project_id":   e.config.ProjectID,
	}
}

// validateTracingConfig checks if the configuration is valid
func validateTracingConfig(c *config.TracingConfig) error {
	if !c.Enabled {
		return nil // No validation needed if disabled
	}

	if c.Endpoint == "" {
		return fmt.Errorf("tracing endpoint is required when tracing is enabled")
	}

	if c.APIKey == "" {
		return fmt.Errorf("W&B API key is required when tracing is enabled")
	}

	if c.ProjectID == "" {
		return fmt.Errorf("W&B project ID is required when tracing is enabled")
	}

	return nil
}
