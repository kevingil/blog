package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type TaskRun struct {
	ID                uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Kind              string         `json:"kind" gorm:"type:varchar(50);not null"`
	TaskName          string         `json:"task_name" gorm:"type:varchar(120);not null;index"`
	Status            string         `json:"status" gorm:"type:varchar(50);not null;index"`
	OrganizationID    *uuid.UUID     `json:"organization_id" gorm:"type:uuid;index"`
	UserID            *uuid.UUID     `json:"user_id" gorm:"type:uuid;index"`
	TriggeredByUserID *uuid.UUID     `json:"triggered_by_user_id" gorm:"type:uuid;index"`
	TriggerSource     string         `json:"trigger_source" gorm:"type:varchar(50);not null;default:'manual'"`
	ParentRunID       *uuid.UUID     `json:"parent_run_id" gorm:"type:uuid;index"`
	Summary           *string        `json:"summary" gorm:"type:text"`
	ErrorSummary      *string        `json:"error_summary" gorm:"type:text"`
	InputPayload      datatypes.JSON `json:"input_payload" gorm:"type:jsonb;default:'{}'"`
	OutputSummary     datatypes.JSON `json:"output_summary" gorm:"type:jsonb;default:'{}'"`
	Metrics           datatypes.JSON `json:"metrics" gorm:"type:jsonb;default:'{}'"`
	StartedAt         *time.Time     `json:"started_at"`
	CompletedAt       *time.Time     `json:"completed_at"`
	CreatedAt         time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt         time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
}

func (TaskRun) TableName() string {
	return "task_run"
}

type TaskRunStep struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	TaskRunID    uuid.UUID      `json:"task_run_id" gorm:"type:uuid;not null;index"`
	StepKey      string         `json:"step_key" gorm:"type:varchar(120);not null"`
	StepName     string         `json:"step_name" gorm:"type:varchar(255);not null"`
	Status       string         `json:"status" gorm:"type:varchar(50);not null;index"`
	Summary      *string        `json:"summary" gorm:"type:text"`
	ErrorSummary *string        `json:"error_summary" gorm:"type:text"`
	Metrics      datatypes.JSON `json:"metrics" gorm:"type:jsonb;default:'{}'"`
	StartedAt    *time.Time     `json:"started_at"`
	CompletedAt  *time.Time     `json:"completed_at"`
	CreatedAt    time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
}

func (TaskRunStep) TableName() string {
	return "task_run_step"
}

type TaskRunEvent struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	TaskRunID     uuid.UUID      `json:"task_run_id" gorm:"type:uuid;not null;index"`
	TaskRunStepID *uuid.UUID     `json:"task_run_step_id" gorm:"type:uuid;index"`
	EventType     string         `json:"event_type" gorm:"type:varchar(120);not null"`
	Level         string         `json:"level" gorm:"type:varchar(20);not null;default:'info'"`
	Message       string         `json:"message" gorm:"type:text;not null"`
	MetaData      datatypes.JSON `json:"meta_data" gorm:"type:jsonb;default:'{}'"`
	CreatedAt     time.Time      `json:"created_at" gorm:"autoCreateTime"`
}

func (TaskRunEvent) TableName() string {
	return "task_run_event"
}
