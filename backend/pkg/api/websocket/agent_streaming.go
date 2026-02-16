// Package websocket provides WebSocket connection management and streaming
package websocket

import (
	"context"
	"encoding/json"
	"log"

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

	writeCount := 0
	for {
		select {
		case response, ok := <-responseChan:
			if !ok {
				log.Printf("[WS] Channel closed after %d writes", writeCount)
				return
			}
			response.RequestID = requestID

			responseBytes, err := json.Marshal(response)
			if err != nil {
				continue
			}
			if err := conn.WriteMessage(websocket.TextMessage, responseBytes); err != nil {
				log.Printf("[WS] Write error after %d writes: %v", writeCount, err)
				return
			}
			writeCount++
			if response.Done || response.Error != "" {
				log.Printf("[WS] Stream done (%d writes, type: %s)", writeCount, response.Type)
				return
			}
		case <-ctx.Done():
			log.Printf("[WS] Context cancelled after %d writes", writeCount)
			return
		}
	}
}
