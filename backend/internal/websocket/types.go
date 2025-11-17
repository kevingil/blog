// Package websocket provides WebSocket connection management and streaming
package websocket

import (
	agentTypes "blog-agent-go/backend/internal/core/agent"
)

// StreamData is an alias for the agent's StreamResponse type
type StreamData = agentTypes.StreamResponse

// SubscribeMessage represents a client subscription message
type SubscribeMessage struct {
	RequestID string `json:"requestId"`
	Action    string `json:"action"`
}

