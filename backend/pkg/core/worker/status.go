package worker

import (
	"sync"
	"time"
)

// WorkerState represents the current state of a worker
type WorkerState string

const (
	StateIdle      WorkerState = "idle"
	StateRunning   WorkerState = "running"
	StateCompleted WorkerState = "completed"
	StateFailed    WorkerState = "failed"
)

// WorkerStatus represents the current status of a worker
type WorkerStatus struct {
	Name        string      `json:"name"`
	State       WorkerState `json:"state"`
	Progress    int         `json:"progress"`     // 0-100
	Message     string      `json:"message"`      // Current operation description
	StartedAt   *time.Time  `json:"started_at"`   // When the current/last run started
	CompletedAt *time.Time  `json:"completed_at"` // When the last run completed
	Error       string      `json:"error"`        // Error message if failed
	ItemsTotal  int         `json:"items_total"`  // Total items to process
	ItemsDone   int         `json:"items_done"`   // Items processed so far
}

// StatusUpdate is sent to subscribers when status changes
type StatusUpdate struct {
	WorkerName string       `json:"worker_name"`
	Status     WorkerStatus `json:"status"`
	Timestamp  time.Time    `json:"timestamp"`
}

// StatusSubscriber is a channel that receives status updates
type StatusSubscriber chan StatusUpdate

// StatusService manages worker status tracking and broadcasting
type StatusService struct {
	statuses    map[string]*WorkerStatus
	subscribers map[StatusSubscriber]struct{}
	mu          sync.RWMutex
}

// Global status service instance
var (
	globalStatusService *StatusService
	statusOnce          sync.Once
)

// GetStatusService returns the global status service instance
func GetStatusService() *StatusService {
	statusOnce.Do(func() {
		globalStatusService = &StatusService{
			statuses:    make(map[string]*WorkerStatus),
			subscribers: make(map[StatusSubscriber]struct{}),
		}
	})
	return globalStatusService
}

// RegisterWorker initializes status tracking for a worker
func (s *StatusService) RegisterWorker(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.statuses[name]; !exists {
		s.statuses[name] = &WorkerStatus{
			Name:  name,
			State: StateIdle,
		}
	}
}

// UpdateStatus updates a worker's status and broadcasts to subscribers
func (s *StatusService) UpdateStatus(name string, state WorkerState, progress int, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	status, exists := s.statuses[name]
	if !exists {
		status = &WorkerStatus{Name: name}
		s.statuses[name] = status
	}

	status.State = state
	status.Progress = progress
	status.Message = message

	if state == StateRunning && status.StartedAt == nil {
		now := time.Now()
		status.StartedAt = &now
		status.CompletedAt = nil
		status.Error = ""
	}

	if state == StateCompleted || state == StateFailed {
		now := time.Now()
		status.CompletedAt = &now
	}

	s.broadcast(name, *status)
}

// SetProgress updates progress with item counts
func (s *StatusService) SetProgress(name string, itemsDone, itemsTotal int, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	status, exists := s.statuses[name]
	if !exists {
		return
	}

	status.ItemsDone = itemsDone
	status.ItemsTotal = itemsTotal
	status.Message = message

	if itemsTotal > 0 {
		status.Progress = (itemsDone * 100) / itemsTotal
	}

	s.broadcast(name, *status)
}

// SetError sets an error message on a worker status
func (s *StatusService) SetError(name string, err string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	status, exists := s.statuses[name]
	if !exists {
		return
	}

	status.Error = err
	status.State = StateFailed
	now := time.Now()
	status.CompletedAt = &now

	s.broadcast(name, *status)
}

// StartWorker marks a worker as starting
func (s *StatusService) StartWorker(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	status, exists := s.statuses[name]
	if !exists {
		status = &WorkerStatus{Name: name}
		s.statuses[name] = status
	}

	now := time.Now()
	status.State = StateRunning
	status.StartedAt = &now
	status.CompletedAt = nil
	status.Progress = 0
	status.Message = "Starting..."
	status.Error = ""
	status.ItemsDone = 0
	status.ItemsTotal = 0

	s.broadcast(name, *status)
}

// CompleteWorker marks a worker as completed
func (s *StatusService) CompleteWorker(name string, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	status, exists := s.statuses[name]
	if !exists {
		return
	}

	now := time.Now()
	status.State = StateCompleted
	status.CompletedAt = &now
	status.Progress = 100
	status.Message = message

	s.broadcast(name, *status)
}

// ResetWorker resets a worker to idle state
func (s *StatusService) ResetWorker(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	status, exists := s.statuses[name]
	if !exists {
		return
	}

	status.State = StateIdle
	status.Progress = 0
	status.Message = ""
	status.ItemsDone = 0
	status.ItemsTotal = 0
	// Keep StartedAt and CompletedAt for "last run" info

	s.broadcast(name, *status)
}

// GetStatus returns the current status of a worker
func (s *StatusService) GetStatus(name string) *WorkerStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if status, exists := s.statuses[name]; exists {
		// Return a copy
		statusCopy := *status
		return &statusCopy
	}
	return nil
}

// GetAllStatuses returns the current status of all workers
func (s *StatusService) GetAllStatuses() map[string]WorkerStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]WorkerStatus, len(s.statuses))
	for name, status := range s.statuses {
		result[name] = *status
	}
	return result
}

// Subscribe adds a subscriber to receive status updates
func (s *StatusService) Subscribe() StatusSubscriber {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(StatusSubscriber, 100) // Buffered to prevent blocking
	s.subscribers[ch] = struct{}{}
	return ch
}

// Unsubscribe removes a subscriber
func (s *StatusService) Unsubscribe(ch StatusSubscriber) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.subscribers, ch)
	close(ch)
}

// broadcast sends a status update to all subscribers (must be called with lock held)
func (s *StatusService) broadcast(name string, status WorkerStatus) {
	update := StatusUpdate{
		WorkerName: name,
		Status:     status,
		Timestamp:  time.Now(),
	}

	for ch := range s.subscribers {
		select {
		case ch <- update:
		default:
			// Channel full, skip this subscriber
		}
	}
}

// GetSubscriberCount returns the number of active subscribers
func (s *StatusService) GetSubscriberCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.subscribers)
}
