// Package metadata provides structured types for message metadata
package metadata

import "time"

// MessageMetaData represents the complete metadata structure for a chat message
type MessageMetaData struct {
	// Artifact information - for content suggestions/changes
	Artifact *ArtifactInfo `json:"artifact,omitempty"`

	// Task/tool execution tracking
	TaskStatus *TaskStatus `json:"task_status,omitempty"`

	// Tool execution details
	ToolExecution *ToolExecution `json:"tool_execution,omitempty"`

	// Context information
	Context *MessageContext `json:"context,omitempty"`

	// User actions (accept/reject)
	UserAction *UserAction `json:"user_action,omitempty"`

	// Attached files/resources
	Attachments []Attachment `json:"attachments,omitempty"`
}

// ArtifactInfo represents a content artifact (edit, rewrite, suggestion)
type ArtifactInfo struct {
	ID          string     `json:"id"`           // Unique artifact ID
	Type        string     `json:"type"`         // "code_edit", "rewrite", "suggestion", "content_generation"
	Status      string     `json:"status"`       // "pending", "accepted", "rejected", "applied"
	Content     string     `json:"content"`      // The actual artifact content
	DiffPreview string     `json:"diff_preview"` // Preview of changes (for edits)
	Title       string     `json:"title"`        // Human-readable artifact title
	Description string     `json:"description"`  // Description of the change
	AppliedAt   *time.Time `json:"applied_at,omitempty"`
}

// Artifact status constants
const (
	ArtifactStatusPending  = "pending"
	ArtifactStatusAccepted = "accepted"
	ArtifactStatusRejected = "rejected"
	ArtifactStatusApplied  = "applied"
)

// Artifact type constants
const (
	ArtifactTypeCodeEdit          = "code_edit"
	ArtifactTypeRewrite           = "rewrite"
	ArtifactTypeSuggestion        = "suggestion"
	ArtifactTypeContentGeneration = "content_generation"
	ArtifactTypeImagePrompt       = "image_prompt"
)

// TaskStatus represents the status of a task or tool execution
type TaskStatus struct {
	TaskID      string     `json:"task_id"`
	Name        string     `json:"name"`
	Status      string     `json:"status"` // "queued", "in_progress", "completed", "failed"
	Progress    float64    `json:"progress"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Error       string     `json:"error,omitempty"`
}

// Task status constants
const (
	TaskStatusQueued     = "queued"
	TaskStatusInProgress = "in_progress"
	TaskStatusCompleted  = "completed"
	TaskStatusFailed     = "failed"
)

// ToolExecution represents details about a tool execution
type ToolExecution struct {
	ToolName   string      `json:"tool_name"`
	ToolID     string      `json:"tool_id"`
	Input      interface{} `json:"input"`
	Output     interface{} `json:"output"`
	Error      string      `json:"error,omitempty"`
	Duration   int64       `json:"duration_ms"` // Duration in milliseconds
	ExecutedAt time.Time   `json:"executed_at"`
	Success    bool        `json:"success"`
}

// MessageContext represents contextual information for a message
type MessageContext struct {
	ArticleID       string `json:"article_id,omitempty"`
	SessionID       string `json:"session_id"`
	RequestID       string `json:"request_id,omitempty"`
	DocumentVersion string `json:"document_version,omitempty"`
	DocumentHash    string `json:"document_hash,omitempty"`
	UserID          string `json:"user_id,omitempty"`
}

// UserAction represents a user's action on an artifact or message
type UserAction struct {
	Action     string    `json:"action"` // "accept", "reject", "modify"
	Timestamp  time.Time `json:"timestamp"`
	ArtifactID string    `json:"artifact_id,omitempty"`
	Feedback   string    `json:"feedback,omitempty"`
	Reason     string    `json:"reason,omitempty"` // For rejections
}

// User action constants
const (
	UserActionAccept = "accept"
	UserActionReject = "reject"
	UserActionModify = "modify"
)

// Attachment represents an attached file or resource
type Attachment struct {
	Type     string `json:"type"` // "file", "link", "image"
	Name     string `json:"name"`
	URL      string `json:"url"`
	MimeType string `json:"mime_type,omitempty"`
	Size     int64  `json:"size,omitempty"`
}

// Attachment type constants
const (
	AttachmentTypeFile  = "file"
	AttachmentTypeLink  = "link"
	AttachmentTypeImage = "image"
)

