// Package websocket provides WebSocket connection management and streaming
package websocket

import (
	agentTypes "backend/pkg/core/agent"
	"time"
)

// StreamData is an alias for the agent's StreamResponse type
type StreamData = agentTypes.StreamResponse

// SubscribeMessage represents a client subscription message
type SubscribeMessage struct {
	RequestID string `json:"requestId"`
	Action    string `json:"action"`
	Channel   string `json:"channel,omitempty"` // For channel-based subscriptions like "worker-status"
}

// WorkerStatusMessage represents a worker status WebSocket message
type WorkerStatusMessage struct {
	Type       string             `json:"type"`       // Always "worker-status"
	WorkerName string             `json:"worker_name"`
	Status     WorkerStatusData   `json:"status"`
	Timestamp  time.Time          `json:"timestamp"`
}

// WorkerStatusData represents worker status in WebSocket messages
type WorkerStatusData struct {
	Name        string  `json:"name"`
	State       string  `json:"state"`
	Progress    int     `json:"progress"`
	Message     string  `json:"message"`
	StartedAt   *string `json:"started_at,omitempty"`
	CompletedAt *string `json:"completed_at,omitempty"`
	Error       string  `json:"error,omitempty"`
	ItemsTotal  int     `json:"items_total"`
	ItemsDone   int     `json:"items_done"`
}

// Channel constants for WebSocket subscriptions
const (
	ChannelWorkerStatus = "worker-status"
)

