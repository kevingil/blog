// Package websocket provides WebSocket connection management and streaming
package websocket

import (
	"context"
	"encoding/json"

	agentTypes "backend/pkg/core/agent"

	"github.com/gofiber/contrib/websocket"
)

// AgentStreamProvider defines the interface for retrieving agent response channels
type AgentStreamProvider interface {
	GetResponseChannel(requestID string) (<-chan agentTypes.StreamResponse, bool)
}

// HandleAgentStream handles streaming for an agent request
func HandleAgentStream(ctx context.Context, conn *websocket.Conn, requestID string, provider AgentStreamProvider) {
	responseChan, exists := provider.GetResponseChannel(requestID)
	if !exists {
		errorMsg := agentTypes.StreamResponse{
			RequestID: requestID,
			Type:      "error",
			Error:     "Request not found",
			Done:      true,
		}
		if msgBytes, err := json.Marshal(errorMsg); err == nil {
			conn.WriteMessage(websocket.TextMessage, msgBytes)
		}
		return
	}
	
	for {
		select {
		case response, ok := <-responseChan:
			if !ok {
				return
			}
			response.RequestID = requestID
			
			responseBytes, err := json.Marshal(response)
			if err != nil {
				continue
			}
			if err := conn.WriteMessage(websocket.TextMessage, responseBytes); err != nil {
				return
			}
			if response.Done || response.Error != "" {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

