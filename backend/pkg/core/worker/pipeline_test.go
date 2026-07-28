package worker

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"
)

type stubWorker struct {
	name  string
	runFn func(ctx context.Context) (*WorkerResult, error)
}

func (w *stubWorker) Name() string {
	return w.name
}

func (w *stubWorker) Run(ctx context.Context) (*WorkerResult, error) {
	if w.runFn != nil {
		return w.runFn(ctx)
	}
	return &WorkerResult{Status: ResultStatusCompleted, Summary: "ok"}, nil
}

func TestPipelineWorkerRunsCrawlThenInsight(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	manager := NewWorkerManager(logger)
	statusService := GetStatusService()

	statusService.ResetWorker("crawl")
	statusService.ResetWorker("insight")
	statusService.ResetWorker(PipelineWorkerName)

	runOrder := make(chan string, 2)

	manager.RegisterWorker(&stubWorker{
		name: "crawl",
		runFn: func(ctx context.Context) (*WorkerResult, error) {
			runOrder <- "crawl"
			time.Sleep(20 * time.Millisecond)
			return &WorkerResult{Status: ResultStatusCompleted, Summary: "crawl ok"}, nil
		},
	})
	manager.RegisterWorker(&stubWorker{
		name: "insight",
		runFn: func(ctx context.Context) (*WorkerResult, error) {
			runOrder <- "insight"
			time.Sleep(20 * time.Millisecond)
			return &WorkerResult{Status: ResultStatusCompleted, Summary: "insight ok"}, nil
		},
	})

	pipeline := NewPipelineWorker(logger, manager)
	manager.RegisterWorker(pipeline)

	if err := manager.RunWorkerNow(PipelineWorkerName); err != nil {
		t.Fatalf("RunWorkerNow returned error: %v", err)
	}

	waitForState(t, statusService, PipelineWorkerName, StateCompleted)

	first := <-runOrder
	second := <-runOrder
	if first != "crawl" || second != "insight" {
		t.Fatalf("expected crawl then insight, got %s then %s", first, second)
	}
}

func TestPipelineWorkerStopsAfterFailedCrawl(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	manager := NewWorkerManager(logger)
	statusService := GetStatusService()

	statusService.ResetWorker("crawl")
	statusService.ResetWorker("insight")
	statusService.ResetWorker(PipelineWorkerName)

	insightStarted := make(chan struct{}, 1)

	manager.RegisterWorker(&stubWorker{
		name: "crawl",
		runFn: func(ctx context.Context) (*WorkerResult, error) {
			time.Sleep(20 * time.Millisecond)
			return nil, errors.New("crawl exploded")
		},
	})
	manager.RegisterWorker(&stubWorker{
		name: "insight",
		runFn: func(ctx context.Context) (*WorkerResult, error) {
			insightStarted <- struct{}{}
			return &WorkerResult{Status: ResultStatusCompleted, Summary: "insight ok"}, nil
		},
	})

	pipeline := NewPipelineWorker(logger, manager)
	manager.RegisterWorker(pipeline)

	if err := manager.RunWorkerNow(PipelineWorkerName); err != nil {
		t.Fatalf("RunWorkerNow returned error: %v", err)
	}

	waitForState(t, statusService, PipelineWorkerName, StateFailed)

	select {
	case <-insightStarted:
		t.Fatal("insight worker should not have started after crawl failure")
	default:
	}
}

func waitForState(t *testing.T, statusService *StatusService, name string, state WorkerState) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		status := statusService.GetStatus(name)
		if status != nil && status.State == state {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	status := statusService.GetStatus(name)
	if status == nil {
		t.Fatalf("worker status %q was never registered", name)
	}

	t.Fatalf("worker %q did not reach state %q, current state %q", name, state, status.State)
}
