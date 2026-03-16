package types

import (
	"time"

	"github.com/google/uuid"
)

type TaskRunKind string

const (
	TaskRunKindWorker   TaskRunKind = "worker"
	TaskRunKindWorkflow TaskRunKind = "workflow"
	TaskRunKindAgent    TaskRunKind = "agent"
)

type TaskRunStatus string

const (
	TaskRunStatusQueued    TaskRunStatus = "queued"
	TaskRunStatusRunning   TaskRunStatus = "running"
	TaskRunStatusCompleted TaskRunStatus = "completed"
	TaskRunStatusWarning   TaskRunStatus = "warning"
	TaskRunStatusFailed    TaskRunStatus = "failed"
	TaskRunStatusCancelled TaskRunStatus = "cancelled"
)

type TaskRunEventLevel string

const (
	TaskRunEventLevelInfo    TaskRunEventLevel = "info"
	TaskRunEventLevelWarning TaskRunEventLevel = "warning"
	TaskRunEventLevelError   TaskRunEventLevel = "error"
)

type TaskRun struct {
	ID                uuid.UUID
	Kind              TaskRunKind
	TaskName          string
	Status            TaskRunStatus
	OrganizationID    *uuid.UUID
	UserID            *uuid.UUID
	TriggeredByUserID *uuid.UUID
	TriggerSource     string
	ParentRunID       *uuid.UUID
	Summary           *string
	ErrorSummary      *string
	InputPayload      map[string]interface{}
	OutputSummary     map[string]interface{}
	Metrics           map[string]interface{}
	StartedAt         *time.Time
	CompletedAt       *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type TaskRunStep struct {
	ID           uuid.UUID
	TaskRunID    uuid.UUID
	StepKey      string
	StepName     string
	Status       TaskRunStatus
	Summary      *string
	ErrorSummary *string
	Metrics      map[string]interface{}
	StartedAt    *time.Time
	CompletedAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type TaskRunEvent struct {
	ID            uuid.UUID
	TaskRunID     uuid.UUID
	TaskRunStepID *uuid.UUID
	EventType     string
	Level         TaskRunEventLevel
	Message       string
	MetaData      map[string]interface{}
	CreatedAt     time.Time
}
