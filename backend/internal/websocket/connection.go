// Package websocket provides WebSocket connection management and streaming
package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
)

const (
	// PingPeriod is how often we send ping messages
	PingPeriod = 30 * time.Second
	// PongWait is how long we wait for pong responses
	PongWait = 60 * time.Second
	// WriteWait is the timeout for writing messages
	WriteWait = 10 * time.Second
)

// Connection represents a WebSocket connection
type Connection struct {
	ID        string
	conn      *websocket.Conn
	send      chan []byte
	requests  map[string]bool // Track subscribed request IDs
	mu        sync.RWMutex
	done      chan struct{}
	closeOnce sync.Once
}

// NewConnection creates a new WebSocket connection
func NewConnection(id string, conn *websocket.Conn) *Connection {
	c := &Connection{
		ID:       id,
		conn:     conn,
		send:     make(chan []byte, 256),
		requests: make(map[string]bool),
		done:     make(chan struct{}),
	}

	// Set up pong handler
	conn.SetReadDeadline(time.Now().Add(PongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(PongWait))
		return nil
	})

	return c
}

// SubscribeToRequest subscribes this connection to a request ID
func (c *Connection) SubscribeToRequest(requestID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requests[requestID] = true
}

// UnsubscribeFromRequest unsubscribes from a request ID
func (c *Connection) UnsubscribeFromRequest(requestID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.requests, requestID)
}

// IsSubscribedTo checks if this connection is subscribed to a request
func (c *Connection) IsSubscribedTo(requestID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.requests[requestID]
}

// Send queues a message to be sent to the client
func (c *Connection) Send(data []byte) {
	select {
	case c.send <- data:
	case <-c.done:
	default:
		// Channel full, skip message to prevent blocking
	}
}

// WritePump handles writing messages to the WebSocket connection
func (c *Connection) WritePump() {
	ticker := time.NewTicker(PingPeriod)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if !ok {
				// Channel closed
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-c.done:
			return
		}
	}
}

// Close closes the connection
func (c *Connection) Close() {
	c.closeOnce.Do(func() {
		close(c.done)
		close(c.send)
		c.conn.Close()
	})
}

// SendError sends an error message to the client
func (c *Connection) SendError(requestID string, errorMsg string) {
	errorData := map[string]interface{}{
		"requestId": requestID,
		"type":      "error",
		"error":     errorMsg,
		"done":      true,
	}
	if data, err := json.Marshal(errorData); err == nil {
		c.Send(data)
	}
}
