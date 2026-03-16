package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

const PipelineWorkerName = "pipeline"

// PipelineWorker orchestrates the crawl -> insight workflow.
type PipelineWorker struct {
	manager       *WorkerManager
	statusService *StatusService
	logger        *slog.Logger
}

// NewPipelineWorker creates a worker that runs the full insights pipeline.
func NewPipelineWorker(logger *slog.Logger, manager *WorkerManager) *PipelineWorker {
	if logger == nil {
		logger = slog.Default()
	}

	return &PipelineWorker{
		manager:       manager,
		statusService: GetStatusService(),
		logger:        logger,
	}
}

// Name returns the worker name.
func (w *PipelineWorker) Name() string {
	return PipelineWorkerName
}

// Run executes the crawl worker and then the insight worker.
func (w *PipelineWorker) Run(ctx context.Context) error {
	if w.manager == nil {
		return fmt.Errorf("worker manager not configured")
	}

	if err := w.runStep(ctx, "crawl", "Crawling sources", 0, 50); err != nil {
		return fmt.Errorf("crawl step failed: %w", err)
	}

	if err := w.runStep(ctx, "insight", "Generating insights", 50, 50); err != nil {
		return fmt.Errorf("insight step failed: %w", err)
	}

	w.statusService.UpdateStatus(w.Name(), StateRunning, 100, "Pipeline complete")
	return nil
}

func (w *PipelineWorker) runStep(
	ctx context.Context,
	workerName string,
	label string,
	baseProgress int,
	progressSpan int,
) error {
	w.statusService.UpdateStatus(w.Name(), StateRunning, baseProgress, label)
	launchedAt := time.Now()

	if err := w.manager.RunWorkerNow(workerName); err != nil {
		return err
	}

	return w.waitForStep(ctx, workerName, label, baseProgress, progressSpan, launchedAt)
}

func (w *PipelineWorker) waitForStep(
	ctx context.Context,
	workerName string,
	label string,
	baseProgress int,
	progressSpan int,
	launchedAt time.Time,
) error {
	subscriber := w.statusService.Subscribe()
	defer w.statusService.Unsubscribe(subscriber)

	done, err := w.handleStepStatus(ctx, workerName, label, baseProgress, progressSpan, launchedAt)
	if err != nil {
		return err
	}
	if done {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			if stopErr := w.manager.StopWorker(workerName); stopErr != nil && stopErr != ErrWorkerNotRunning {
				w.logger.Warn("failed to stop pipeline step after cancellation", "worker", workerName, "error", stopErr)
			}
			return ctx.Err()
		case update, ok := <-subscriber:
			if !ok || update.WorkerName != workerName {
				continue
			}

			done, err := w.handleStepStatusFromValue(workerName, label, baseProgress, progressSpan, launchedAt, update.Status)
			if err != nil {
				return err
			}
			if done {
				return nil
			}
		}
	}
}

func (w *PipelineWorker) handleStepStatus(
	ctx context.Context,
	workerName string,
	label string,
	baseProgress int,
	progressSpan int,
	launchedAt time.Time,
) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	status := w.statusService.GetStatus(workerName)
	if status == nil {
		return false, nil
	}

	return w.handleStepStatusFromValue(workerName, label, baseProgress, progressSpan, launchedAt, *status)
}

func (w *PipelineWorker) handleStepStatusFromValue(
	workerName string,
	label string,
	baseProgress int,
	progressSpan int,
	launchedAt time.Time,
	status WorkerStatus,
) (bool, error) {
	if !isFreshStepStatus(status, launchedAt) {
		return false, nil
	}

	progress := baseProgress + (status.Progress * progressSpan / 100)
	message := label
	if status.Message != "" {
		message = fmt.Sprintf("%s: %s", label, status.Message)
	}

	switch status.State {
	case StateRunning:
		w.statusService.UpdateStatus(w.Name(), StateRunning, progress, message)
	case StateCompleted:
		w.statusService.UpdateStatus(w.Name(), StateRunning, baseProgress+progressSpan, fmt.Sprintf("%s complete", label))
		return true, nil
	case StateFailed:
		errMessage := status.Error
		if errMessage == "" {
			errMessage = status.Message
		}
		if errMessage == "" {
			errMessage = "step failed"
		}
		w.statusService.SetError(w.Name(), fmt.Sprintf("%s failed: %s", label, errMessage))
		return false, fmt.Errorf("%s", errMessage)
	}

	return false, nil
}

func isFreshStepStatus(status WorkerStatus, launchedAt time.Time) bool {
	if status.State == StateIdle {
		return false
	}

	if status.StartedAt == nil {
		return status.State == StateRunning
	}

	return !status.StartedAt.Before(launchedAt)
}
