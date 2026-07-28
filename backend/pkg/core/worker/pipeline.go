package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"backend/pkg/core/taskrun"
	"backend/pkg/types"

	"github.com/google/uuid"
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
func (w *PipelineWorker) Run(ctx context.Context) (*WorkerResult, error) {
	if w.manager == nil {
		return nil, fmt.Errorf("worker manager not configured")
	}

	crawlStatus, err := w.runStep(ctx, "crawl", "Crawling sources", 0, 50)
	if err != nil {
		return nil, fmt.Errorf("crawl step failed: %w", err)
	}

	insightStatus, err := w.runStep(ctx, "insight", "Generating insights", 50, 50)
	if err != nil {
		return nil, fmt.Errorf("insight step failed: %w", err)
	}

	w.statusService.UpdateStatus(w.Name(), StateRunning, 100, "Pipeline complete")
	status := ResultStatusCompleted
	warnings := []string{}
	if crawlStatus == types.TaskRunStatusWarning {
		status = ResultStatusWarning
		warnings = append(warnings, "Crawl completed with warnings")
	}
	if insightStatus == types.TaskRunStatusWarning {
		status = ResultStatusWarning
		warnings = append(warnings, "Insight generation completed with warnings")
	}
	summary := "Pipeline completed successfully"
	if status == ResultStatusWarning {
		summary = "Pipeline completed with warnings"
	}
	return &WorkerResult{
		Status:   status,
		Summary:  summary,
		Metrics:  map[string]interface{}{"crawl_status": crawlStatus, "insight_status": insightStatus},
		Warnings: warnings,
	}, nil
}

func (w *PipelineWorker) runStep(
	ctx context.Context,
	workerName string,
	label string,
	baseProgress int,
	progressSpan int,
) (types.TaskRunStatus, error) {
	w.statusService.UpdateStatus(w.Name(), StateRunning, baseProgress, label)
	launchedAt := time.Now()
	tracker := taskrun.FromContext(ctx)
	if tracker != nil {
		_ = tracker.StartStep(ctx, workerName, workerName, strPtr(label))
	}
	metadata := RunMetadata{
		TriggerSource: "workflow",
	}
	if tracker != nil {
		runID := tracker.RunID()
		metadata.ParentRunID = &runID
	}

	childRunID, err := w.manager.RunWorkerNowWithMetadata(workerName, metadata)
	if err != nil {
		if tracker != nil {
			errMessage := err.Error()
			_ = tracker.FinishStep(ctx, workerName, "failed", strPtr(label+" failed"), &errMessage, nil)
		}
		return types.TaskRunStatusFailed, err
	}

	status, err := w.waitForStep(ctx, workerName, label, baseProgress, progressSpan, launchedAt, childRunID)
	if tracker != nil {
		summary := label + " complete"
		if status == types.TaskRunStatusWarning {
			summary = label + " completed with warnings"
		}
		if err != nil {
			errMessage := err.Error()
			_ = tracker.FinishStep(ctx, workerName, "failed", strPtr(label+" failed"), &errMessage, nil)
		} else {
			_ = tracker.FinishStep(ctx, workerName, string(status), &summary, nil, map[string]interface{}{"child_run_id": childRunID})
		}
	}
	return status, err
}

func (w *PipelineWorker) waitForStep(
	ctx context.Context,
	workerName string,
	label string,
	baseProgress int,
	progressSpan int,
	launchedAt time.Time,
	childRunID string,
) (types.TaskRunStatus, error) {
	subscriber := w.statusService.Subscribe()
	defer w.statusService.Unsubscribe(subscriber)

	done, err := w.handleStepStatus(ctx, workerName, label, baseProgress, progressSpan, launchedAt, childRunID)
	if err != nil {
		return types.TaskRunStatusFailed, err
	}
	if done {
		return w.lookupRunStatus(ctx, childRunID), nil
	}

	for {
		select {
		case <-ctx.Done():
			if stopErr := w.manager.StopWorker(workerName); stopErr != nil && stopErr != ErrWorkerNotRunning {
				w.logger.Warn("failed to stop pipeline step after cancellation", "worker", workerName, "error", stopErr)
			}
			return types.TaskRunStatusCancelled, ctx.Err()
		case update, ok := <-subscriber:
			if !ok || update.WorkerName != workerName {
				continue
			}

			done, err := w.handleStepStatusFromValue(workerName, label, baseProgress, progressSpan, launchedAt, update.Status)
			if err != nil {
				return types.TaskRunStatusFailed, err
			}
			if done {
				return w.lookupRunStatus(ctx, childRunID), nil
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
	childRunID string,
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

func (w *PipelineWorker) lookupRunStatus(ctx context.Context, childRunID string) types.TaskRunStatus {
	if childRunID == "" || w.manager.taskRunService == nil {
		return types.TaskRunStatusCompleted
	}
	runUUID, err := uuid.Parse(childRunID)
	if err != nil {
		return types.TaskRunStatusCompleted
	}
	run, err := w.manager.taskRunService.GetRun(ctx, runUUID)
	if err != nil {
		return types.TaskRunStatusCompleted
	}
	return run.Status
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

func strPtr(value string) *string {
	return &value
}
