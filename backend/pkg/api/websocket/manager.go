// Package websocket provides WebSocket connection management and streaming
package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"backend/pkg/core/worker"

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
	connections      map[string]*Connection
	channelSubscribers map[string]map[string]bool // channel -> connectionID -> subscribed
	mu               sync.RWMutex
	config           Config
	stopWorkerStatus chan struct{}
}

// NewManager creates a new WebSocket manager
func NewManager(config Config) *Manager {
	return &Manager{
		connections:      make(map[string]*Connection),
		channelSubscribers: make(map[string]map[string]bool),
		config:           config,
		stopWorkerStatus: make(chan struct{}),
	}
}

// StartWorkerStatusBroadcast starts listening to worker status updates and broadcasting them
func (m *Manager) StartWorkerStatusBroadcast() {
	statusService := worker.GetStatusService()
	subscriber := statusService.Subscribe()

	go func() {
		defer statusService.Unsubscribe(subscriber)

		for {
			select {
			case update, ok := <-subscriber:
				if !ok {
					return
				}
				m.broadcastWorkerStatus(update)
			case <-m.stopWorkerStatus:
				return
			}
		}
	}()
}

// StopWorkerStatusBroadcast stops the worker status broadcast
func (m *Manager) StopWorkerStatusBroadcast() {
	close(m.stopWorkerStatus)
}

// broadcastWorkerStatus broadcasts a worker status update to all subscribed connections
func (m *Manager) broadcastWorkerStatus(update worker.StatusUpdate) {
	// Format timestamps
	var startedAt, completedAt *string
	if update.Status.StartedAt != nil {
		s := update.Status.StartedAt.Format(time.RFC3339)
		startedAt = &s
	}
	if update.Status.CompletedAt != nil {
		s := update.Status.CompletedAt.Format(time.RFC3339)
		completedAt = &s
	}

	msg := WorkerStatusMessage{
		Type:       "worker-status",
		WorkerName: update.WorkerName,
		Status: WorkerStatusData{
			Name:        update.Status.Name,
			State:       string(update.Status.State),
			Progress:    update.Status.Progress,
			Message:     update.Status.Message,
			StartedAt:   startedAt,
			CompletedAt: completedAt,
			Error:       update.Status.Error,
			ItemsTotal:  update.Status.ItemsTotal,
			ItemsDone:   update.Status.ItemsDone,
		},
		Timestamp: update.Timestamp,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[WebSocket] Failed to marshal worker status: %v", err)
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	subscribers, exists := m.channelSubscribers[ChannelWorkerStatus]
	if !exists {
		return
	}

	for connID := range subscribers {
		if conn, ok := m.connections[connID]; ok {
			conn.Send(data)
		}
	}
}

// subscribeToChannel subscribes a connection to a channel
func (m *Manager) subscribeToChannel(connID, channel string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.channelSubscribers[channel]; !exists {
		m.channelSubscribers[channel] = make(map[string]bool)
	}
	m.channelSubscribers[channel][connID] = true
}

// unsubscribeFromChannel unsubscribes a connection from a channel
func (m *Manager) unsubscribeFromChannel(connID, channel string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if subs, exists := m.channelSubscribers[channel]; exists {
		delete(subs, connID)
	}
}

// unsubscribeFromAllChannels unsubscribes a connection from all channels
func (m *Manager) unsubscribeFromAllChannels(connID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, subs := range m.channelSubscribers {
		delete(subs, connID)
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
	m.unsubscribeFromAllChannels(connectionID)
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

			// Handle channel-based subscriptions (e.g., worker-status)
			if msg.Channel != "" {
				if msg.Action == "subscribe" {
					m.subscribeToChannel(conn.ID, msg.Channel)
					// Send acknowledgment
					ack := map[string]interface{}{
						"type":    "subscribed",
						"channel": msg.Channel,
					}
					if data, err := json.Marshal(ack); err == nil {
						conn.Send(data)
					}
				} else if msg.Action == "unsubscribe" {
					m.unsubscribeFromChannel(conn.ID, msg.Channel)
				}
				continue
			}

			// Handle request-based subscriptions (existing behavior)
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

