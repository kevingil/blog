package worker

import "github.com/google/uuid"

type ResultStatus string

const (
	ResultStatusCompleted ResultStatus = "completed"
	ResultStatusWarning   ResultStatus = "warning"
)

type WorkerResult struct {
	Status        ResultStatus
	Summary       string
	OutputSummary map[string]interface{}
	Metrics       map[string]interface{}
	Warnings      []string
}

type RunMetadata struct {
	UserID            *uuid.UUID
	OrganizationID    *uuid.UUID
	TriggeredByUserID *uuid.UUID
	ParentRunID       *uuid.UUID
	TriggerSource     string
}
