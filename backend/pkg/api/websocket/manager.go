// Package websocket provides WebSocket connection management and streaming
package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/gofiber/contrib/websocket"
	"github.com/google/uuid"
)

// Config holds WebSocket manager configuration
type Config struct {
	MaxConnections int
}

// DefaultConfig returns the default WebSocket configuration
func DefaultConfig() Config {
	return Config{
		MaxConnections: 1000,
	}
}

// Manager manages WebSocket connections and streaming
type Manager struct {
	connections map[string]*Connection
	mu          sync.RWMutex
	config      Config
}

// NewManager creates a new WebSocket manager
func NewManager(config Config) *Manager {
	return &Manager{
		connections: make(map[string]*Connection),
		config:      config,
	}
}

// HandleConnection handles a new WebSocket connection
func (m *Manager) HandleConnection(conn *websocket.Conn) {
	m.mu.Lock()
	if len(m.connections) >= m.config.MaxConnections {
		m.mu.Unlock()
		conn.Close()
		return
	}

	connectionID := uuid.New().String()
	connection := NewConnection(connectionID, conn)
	m.connections[connectionID] = connection
	m.mu.Unlock()

	// Start write pump
	go connection.WritePump()

	// Handle incoming messages
	m.readPump(connection)

	// Cleanup on disconnect
	m.mu.Lock()
	delete(m.connections, connectionID)
	m.mu.Unlock()
	connection.Close()
}

// readPump handles reading messages from the WebSocket connection
func (m *Manager) readPump(conn *Connection) {
	defer conn.Close()

	for {
		messageType, message, err := conn.conn.ReadMessage()
		if err != nil {
			break
		}

		if messageType == websocket.TextMessage {
			var msg SubscribeMessage
			if err := json.Unmarshal(message, &msg); err != nil {
				continue
			}

			if msg.Action == "subscribe" && msg.RequestID != "" {
				conn.SubscribeToRequest(msg.RequestID)
			} else if msg.Action == "unsubscribe" && msg.RequestID != "" {
				conn.UnsubscribeFromRequest(msg.RequestID)
			}
		}
	}
}

// StreamToRequest sends data to all connections subscribed to a request
func (m *Manager) StreamToRequest(requestID string, data StreamData) {
	data.RequestID = requestID
	
	dataBytes, err := json.Marshal(data)
	if err != nil {
		log.Printf("[WebSocket] Failed to marshal data for request %s: %v", requestID, err)
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, conn := range m.connections {
		if conn.IsSubscribedTo(requestID) {
			conn.Send(dataBytes)
		}
	}
}

// StreamToRequestContext streams data to a request with context for cancellation
func (m *Manager) StreamToRequestContext(ctx context.Context, requestID string, dataChan <-chan StreamData) {
	for {
		select {
		case data, ok := <-dataChan:
			if !ok {
				return
			}
			m.StreamToRequest(requestID, data)
			if data.Done || data.Error != "" {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// Close closes all connections
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, conn := range m.connections {
		conn.Close()
	}
	m.connections = make(map[string]*Connection)
}

// ActiveConnections returns the number of active connections
func (m *Manager) ActiveConnections() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.connections)
}

