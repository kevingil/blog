package tracing

import (
	"time"
)

// W&B Weave Call API Types
// Based on https://trace.wandb.ai/call/start

type CallStartRequest struct {
	Start CallStart `json:"start"`
}

type CallEndRequest struct {
	End CallEnd `json:"end"`
}

type CallStart struct {
	ProjectID   string                 `json:"project_id"`
	ID          string                 `json:"id"`
	OpName      string                 `json:"op_name"`
	DisplayName string                 `json:"display_name,omitempty"`
	TraceID     string                 `json:"trace_id"`
	ParentID    string                 `json:"parent_id,omitempty"`
	StartedAt   string                 `json:"started_at"`
	Attributes  map[string]interface{} `json:"attributes,omitempty"`
	Inputs      map[string]interface{} `json:"inputs,omitempty"`
	WBUserID    string                 `json:"wb_user_id,omitempty"`
	WBRunID     string                 `json:"wb_run_id,omitempty"`
}

type CallEnd struct {
	ProjectID string                 `json:"project_id"`
	ID        string                 `json:"id"`
	EndedAt   string                 `json:"ended_at"`
	Outputs   map[string]interface{} `json:"outputs,omitempty"`
	Exception string                 `json:"exception,omitempty"`
}

// Helper functions for W&B Weave API
func TimeToRFC3339(t time.Time) string {
	return t.Format(time.RFC3339)
}

// Simple attribute helper functions
func StringAttr(key, value string) map[string]interface{} {
	return map[string]interface{}{key: value}
}

func IntAttr(key string, value int64) map[string]interface{} {
	return map[string]interface{}{key: value}
}

func BoolAttr(key string, value bool) map[string]interface{} {
	return map[string]interface{}{key: value}
}

func FloatAttr(key string, value float64) map[string]interface{} {
	return map[string]interface{}{key: value}
}
