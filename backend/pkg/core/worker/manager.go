package worker

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// Error types for worker operations
var (
	ErrWorkerNotFound       = errors.New("worker not found")
	ErrWorkerAlreadyRunning = errors.New("worker is already running")
	ErrWorkerNotRunning     = errors.New("worker is not running")
)

// Worker interface defines the contract for background workers
type Worker interface {
	Name() string
	Run(ctx context.Context) error
}

// WorkerManager manages background workers and their scheduling
type WorkerManager struct {
	cron            *cron.Cron
	workers         map[string]Worker
	runningWorkers  map[string]context.CancelFunc
	mu              sync.RWMutex
	logger          *slog.Logger
	isRunning       bool
	shutdownTimeout time.Duration
	statusService   *StatusService
}

// NewWorkerManager creates a new WorkerManager instance
func NewWorkerManager(logger *slog.Logger) *WorkerManager {
	if logger == nil {
		logger = slog.Default()
	}

	return &WorkerManager{
		cron:            cron.New(cron.WithSeconds()),
		workers:         make(map[string]Worker),
		runningWorkers:  make(map[string]context.CancelFunc),
		logger:          logger,
		shutdownTimeout: 30 * time.Second,
		statusService:   GetStatusService(),
	}
}

// RegisterWorker registers a worker with the manager
func (m *WorkerManager) RegisterWorker(worker Worker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.workers[worker.Name()] = worker
	m.statusService.RegisterWorker(worker.Name())
	m.logger.Info("registered worker", "name", worker.Name())
}

// ScheduleWorker schedules a worker to run on a cron schedule
// Schedule format: "seconds minutes hours day-of-month month day-of-week"
// Examples:
// - "0 */5 * * * *" - every 5 minutes
// - "0 0 * * * *" - every hour
// - "0 0 9 * * *" - every day at 9 AM
func (m *WorkerManager) ScheduleWorker(workerName, schedule string) error {
	m.mu.RLock()
	worker, exists := m.workers[workerName]
	m.mu.RUnlock()

	if !exists {
		m.logger.Error("worker not found", "name", workerName)
		return nil
	}

	_, err := m.cron.AddFunc(schedule, func() {
		m.runWorker(worker)
	})

	if err != nil {
		m.logger.Error("failed to schedule worker", "name", workerName, "error", err)
		return err
	}

	m.logger.Info("scheduled worker", "name", workerName, "schedule", schedule)
	return nil
}

// runWorker runs a worker with proper context management
func (m *WorkerManager) runWorker(worker Worker) {
	name := worker.Name()

	// Check if worker is already running
	m.mu.Lock()
	if _, running := m.runningWorkers[name]; running {
		m.mu.Unlock()
		m.logger.Warn("worker already running, skipping", "name", name)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.runningWorkers[name] = cancel
	m.mu.Unlock()

	m.logger.Info("starting worker", "name", name)
	m.statusService.StartWorker(name)
	startTime := time.Now()

	defer func() {
		m.mu.Lock()
		delete(m.runningWorkers, name)
		m.mu.Unlock()

		duration := time.Since(startTime)
		m.logger.Info("worker completed", "name", name, "duration", duration)
	}()

	if err := worker.Run(ctx); err != nil {
		m.logger.Error("worker failed", "name", name, "error", err)
		m.statusService.SetError(name, err.Error())
	} else {
		m.statusService.CompleteWorker(name, "Completed successfully")
	}
}

// RunWorkerNow runs a worker immediately (outside of schedule)
func (m *WorkerManager) RunWorkerNow(workerName string) error {
	m.mu.RLock()
	worker, exists := m.workers[workerName]
	_, isRunning := m.runningWorkers[workerName]
	m.mu.RUnlock()

	if !exists {
		m.logger.Error("worker not found", "name", workerName)
		return ErrWorkerNotFound
	}

	if isRunning {
		m.logger.Warn("worker already running", "name", workerName)
		return ErrWorkerAlreadyRunning
	}

	go m.runWorker(worker)
	return nil
}

// StopWorker stops a running worker
func (m *WorkerManager) StopWorker(workerName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cancel, running := m.runningWorkers[workerName]
	if !running {
		return ErrWorkerNotRunning
	}

	m.logger.Info("stopping worker", "name", workerName)
	cancel()
	m.statusService.SetError(workerName, "Stopped by user")
	return nil
}

// IsWorkerRunning checks if a specific worker is currently running
func (m *WorkerManager) IsWorkerRunning(workerName string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, running := m.runningWorkers[workerName]
	return running
}

// Start starts the cron scheduler
func (m *WorkerManager) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		return
	}

	m.cron.Start()
	m.isRunning = true
	m.logger.Info("worker manager started")
}

// Stop stops the cron scheduler and waits for running workers to finish
func (m *WorkerManager) Stop() {
	m.mu.Lock()
	if !m.isRunning {
		m.mu.Unlock()
		return
	}

	// Stop accepting new jobs
	ctx := m.cron.Stop()
	m.isRunning = false

	// Cancel all running workers
	for name, cancel := range m.runningWorkers {
		m.logger.Info("cancelling worker", "name", name)
		cancel()
	}
	m.mu.Unlock()

	// Wait for cron to finish (with timeout)
	select {
	case <-ctx.Done():
		m.logger.Info("all cron jobs completed")
	case <-time.After(m.shutdownTimeout):
		m.logger.Warn("shutdown timeout exceeded, some workers may not have completed")
	}

	m.logger.Info("worker manager stopped")
}

// IsRunning returns whether the worker manager is running
func (m *WorkerManager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isRunning
}

// GetRunningWorkers returns the names of currently running workers
func (m *WorkerManager) GetRunningWorkers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.runningWorkers))
	for name := range m.runningWorkers {
		names = append(names, name)
	}
	return names
}

// GetRegisteredWorkers returns the names of all registered workers
func (m *WorkerManager) GetRegisteredWorkers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.workers))
	for name := range m.workers {
		names = append(names, name)
	}
	return names
}

// Global worker manager instance for API access
var globalManager *WorkerManager

// SetGlobalManager sets the global worker manager instance
func SetGlobalManager(m *WorkerManager) {
	globalManager = m
}

// GetManager returns the global worker manager instance
func GetManager() *WorkerManager {
	return globalManager
}
