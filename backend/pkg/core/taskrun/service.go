package taskrun

import (
	"context"
	"time"

	"backend/pkg/database/repository"
	"backend/pkg/types"

	"github.com/google/uuid"
)

type Service struct {
	repo repository.TaskRunRepository
}

func NewService(repo repository.TaskRunRepository) *Service {
	return &Service{repo: repo}
}

type StartRunInput struct {
	Kind              types.TaskRunKind
	TaskName          string
	OrganizationID    *uuid.UUID
	UserID            *uuid.UUID
	TriggeredByUserID *uuid.UUID
	TriggerSource     string
	ParentRunID       *uuid.UUID
	InputPayload      map[string]interface{}
	Summary           *string
}

type FinishRunInput struct {
	RunID          uuid.UUID
	Status         types.TaskRunStatus
	Summary        *string
	ErrorSummary   *string
	OutputSummary  map[string]interface{}
	Metrics        map[string]interface{}
}

type StartStepInput struct {
	RunID     uuid.UUID
	StepKey   string
	StepName  string
	Summary   *string
}

type FinishStepInput struct {
	RunID        uuid.UUID
	StepKey      string
	Status       types.TaskRunStatus
	Summary      *string
	ErrorSummary *string
	Metrics      map[string]interface{}
}

type RecordEventInput struct {
	RunID     uuid.UUID
	StepKey   *string
	EventType string
	Level     types.TaskRunEventLevel
	Message   string
	MetaData  map[string]interface{}
}

func (s *Service) StartRun(ctx context.Context, input StartRunInput) (*types.TaskRun, error) {
	now := time.Now()
	run := &types.TaskRun{
		Kind:              input.Kind,
		TaskName:          input.TaskName,
		Status:            types.TaskRunStatusRunning,
		OrganizationID:    input.OrganizationID,
		UserID:            input.UserID,
		TriggeredByUserID: input.TriggeredByUserID,
		TriggerSource:     input.TriggerSource,
		ParentRunID:       input.ParentRunID,
		InputPayload:      input.InputPayload,
		OutputSummary:     map[string]interface{}{},
		Metrics:           map[string]interface{}{},
		StartedAt:         &now,
		Summary:           input.Summary,
	}
	if run.TriggerSource == "" {
		run.TriggerSource = "manual"
	}
	if err := s.repo.CreateRun(ctx, run); err != nil {
		return nil, err
	}
	return run, s.RecordEvent(ctx, RecordEventInput{
		RunID:     run.ID,
		EventType: "run_started",
		Level:     types.TaskRunEventLevelInfo,
		Message:   "Run started",
	})
}

func (s *Service) FinishRun(ctx context.Context, input FinishRunInput) error {
	run, err := s.repo.FindRunByID(ctx, input.RunID)
	if err != nil {
		return err
	}

	now := time.Now()
	run.Status = input.Status
	run.Summary = input.Summary
	run.ErrorSummary = input.ErrorSummary
	run.OutputSummary = input.OutputSummary
	run.Metrics = input.Metrics
	run.CompletedAt = &now

	if err := s.repo.UpdateRun(ctx, run); err != nil {
		return err
	}

	eventType := "run_completed"
	level := types.TaskRunEventLevelInfo
	message := "Run completed"
	if input.Status == types.TaskRunStatusWarning {
		eventType = "run_warning"
		level = types.TaskRunEventLevelWarning
		message = "Run completed with warnings"
	} else if input.Status == types.TaskRunStatusFailed {
		eventType = "run_failed"
		level = types.TaskRunEventLevelError
		message = "Run failed"
	} else if input.Status == types.TaskRunStatusCancelled {
		eventType = "run_cancelled"
		level = types.TaskRunEventLevelWarning
		message = "Run cancelled"
	}

	if run.Summary != nil && *run.Summary != "" {
		message = *run.Summary
	}

	return s.RecordEvent(ctx, RecordEventInput{
		RunID:     run.ID,
		EventType: eventType,
		Level:     level,
		Message:   message,
		MetaData: map[string]interface{}{
			"status": input.Status,
		},
	})
}

func (s *Service) StartStep(ctx context.Context, input StartStepInput) (*types.TaskRunStep, error) {
	now := time.Now()
	step := &types.TaskRunStep{
		TaskRunID: input.RunID,
		StepKey:   input.StepKey,
		StepName:  input.StepName,
		Status:    types.TaskRunStatusRunning,
		Summary:   input.Summary,
		StartedAt: &now,
		Metrics:   map[string]interface{}{},
	}
	if err := s.repo.CreateStep(ctx, step); err != nil {
		return nil, err
	}
	return step, s.RecordEvent(ctx, RecordEventInput{
		RunID:     input.RunID,
		StepKey:   &input.StepKey,
		EventType: "step_started",
		Level:     types.TaskRunEventLevelInfo,
		Message:   input.StepName,
	})
}

func (s *Service) FinishStep(ctx context.Context, input FinishStepInput) error {
	step, err := s.repo.FindStepByRunAndKey(ctx, input.RunID, input.StepKey)
	if err != nil {
		return err
	}

	now := time.Now()
	step.Status = input.Status
	step.Summary = input.Summary
	step.ErrorSummary = input.ErrorSummary
	step.Metrics = input.Metrics
	step.CompletedAt = &now

	if err := s.repo.UpdateStep(ctx, step); err != nil {
		return err
	}

	level := types.TaskRunEventLevelInfo
	eventType := "step_completed"
	message := step.StepName
	if input.Status == types.TaskRunStatusWarning {
		level = types.TaskRunEventLevelWarning
		eventType = "step_warning"
		message = step.StepName + " completed with warnings"
	} else if input.Status == types.TaskRunStatusFailed {
		level = types.TaskRunEventLevelError
		eventType = "step_failed"
		message = step.StepName + " failed"
	}
	if input.Summary != nil && *input.Summary != "" {
		message = *input.Summary
	}

	return s.RecordEvent(ctx, RecordEventInput{
		RunID:     input.RunID,
		StepKey:   &input.StepKey,
		EventType: eventType,
		Level:     level,
		Message:   message,
		MetaData:  input.Metrics,
	})
}

func (s *Service) RecordEvent(ctx context.Context, input RecordEventInput) error {
	var stepID *uuid.UUID
	if input.StepKey != nil {
		step, err := s.repo.FindStepByRunAndKey(ctx, input.RunID, *input.StepKey)
		if err == nil {
			stepID = &step.ID
		}
	}

	return s.repo.CreateEvent(ctx, &types.TaskRunEvent{
		TaskRunID:     input.RunID,
		TaskRunStepID: stepID,
		EventType:     input.EventType,
		Level:         input.Level,
		Message:       input.Message,
		MetaData:      input.MetaData,
	})
}

func (s *Service) GetRun(ctx context.Context, id uuid.UUID) (*types.TaskRun, error) {
	return s.repo.FindRunByID(ctx, id)
}

func (s *Service) ListRuns(ctx context.Context, filter repository.TaskRunFilter) ([]types.TaskRun, error) {
	return s.repo.ListRuns(ctx, filter)
}

func (s *Service) ListStepsByRunID(ctx context.Context, runID uuid.UUID) ([]types.TaskRunStep, error) {
	return s.repo.ListStepsByRunID(ctx, runID)
}

func (s *Service) ListEventsByRunID(ctx context.Context, runID uuid.UUID) ([]types.TaskRunEvent, error) {
	return s.repo.ListEventsByRunID(ctx, runID)
}
