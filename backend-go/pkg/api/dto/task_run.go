package dto

type TaskRunResponse struct {
	ID             string                 `json:"id"`
	Kind           string                 `json:"kind"`
	TaskName       string                 `json:"task_name"`
	Status         string                 `json:"status"`
	TriggerSource  string                 `json:"trigger_source"`
	Summary        *string                `json:"summary,omitempty"`
	ErrorSummary   *string                `json:"error_summary,omitempty"`
	StartedAt      *string                `json:"started_at,omitempty"`
	CompletedAt    *string                `json:"completed_at,omitempty"`
	DurationMS     *int64                 `json:"duration_ms,omitempty"`
	OutputSummary  map[string]interface{} `json:"output_summary,omitempty"`
	Metrics        map[string]interface{} `json:"metrics,omitempty"`
	ParentRunID    *string                `json:"parent_run_id,omitempty"`
}

type TaskRunStepResponse struct {
	ID           string                 `json:"id"`
	StepKey      string                 `json:"step_key"`
	StepName     string                 `json:"step_name"`
	Status       string                 `json:"status"`
	Summary      *string                `json:"summary,omitempty"`
	ErrorSummary *string                `json:"error_summary,omitempty"`
	StartedAt    *string                `json:"started_at,omitempty"`
	CompletedAt  *string                `json:"completed_at,omitempty"`
	Metrics      map[string]interface{} `json:"metrics,omitempty"`
}

type TaskRunEventResponse struct {
	ID        string                 `json:"id"`
	EventType string                 `json:"event_type"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	CreatedAt string                 `json:"created_at"`
	StepKey   *string                `json:"step_key,omitempty"`
	MetaData  map[string]interface{} `json:"meta_data,omitempty"`
}

type TaskRunListResponse struct {
	Runs []TaskRunResponse `json:"runs"`
}

type TaskRunDetailResponse struct {
	Run    TaskRunResponse        `json:"run"`
	Steps  []TaskRunStepResponse  `json:"steps"`
	Events []TaskRunEventResponse `json:"events"`
}
