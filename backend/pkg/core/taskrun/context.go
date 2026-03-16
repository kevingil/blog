package taskrun

import (
	"context"

	"backend/pkg/types"

	"github.com/google/uuid"
)

type trackerContextKey struct{}

type Tracker struct {
	service *Service
	runID   uuid.UUID
}

func NewTracker(service *Service, runID uuid.UUID) *Tracker {
	if service == nil || runID == uuid.Nil {
		return nil
	}
	return &Tracker{service: service, runID: runID}
}

func WithTracker(ctx context.Context, tracker *Tracker) context.Context {
	if tracker == nil {
		return ctx
	}
	return context.WithValue(ctx, trackerContextKey{}, tracker)
}

func FromContext(ctx context.Context) *Tracker {
	tracker, _ := ctx.Value(trackerContextKey{}).(*Tracker)
	return tracker
}

func (t *Tracker) RunID() uuid.UUID {
	if t == nil {
		return uuid.Nil
	}
	return t.runID
}

func (t *Tracker) StartStep(ctx context.Context, stepKey, stepName string, summary *string) error {
	if t == nil {
		return nil
	}
	_, err := t.service.StartStep(ctx, StartStepInput{
		RunID:    t.runID,
		StepKey:  stepKey,
		StepName: stepName,
		Summary:  summary,
	})
	return err
}

func (t *Tracker) FinishStep(ctx context.Context, stepKey string, status string, summary *string, errorSummary *string, metrics map[string]interface{}) error {
	if t == nil {
		return nil
	}
	return t.service.FinishStep(ctx, FinishStepInput{
		RunID:        t.runID,
		StepKey:      stepKey,
		Status:       parseStatus(status),
		Summary:      summary,
		ErrorSummary: errorSummary,
		Metrics:      metrics,
	})
}

func (t *Tracker) RecordEvent(ctx context.Context, stepKey *string, eventType string, level string, message string, meta map[string]interface{}) error {
	if t == nil {
		return nil
	}
	return t.service.RecordEvent(ctx, RecordEventInput{
		RunID:     t.runID,
		StepKey:   stepKey,
		EventType: eventType,
		Level:     parseLevel(level),
		Message:   message,
		MetaData:  meta,
	})
}

func parseStatus(status string) types.TaskRunStatus {
	switch status {
	case "warning":
		return types.TaskRunStatusWarning
	case "failed":
		return types.TaskRunStatusFailed
	case "cancelled":
		return types.TaskRunStatusCancelled
	default:
		return types.TaskRunStatusCompleted
	}
}

func parseLevel(level string) types.TaskRunEventLevel {
	switch level {
	case "warning":
		return types.TaskRunEventLevelWarning
	case "error":
		return types.TaskRunEventLevelError
	default:
		return types.TaskRunEventLevelInfo
	}
}
