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

	// Chain of thought reasoning (from reasoning models)
	Thinking *ThinkingBlock `json:"thinking,omitempty"`
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

// =============================================================================
// Turn-based Message Types (New Architecture)
// =============================================================================

// TurnMetaData represents the complete metadata for an agent turn
// A turn is a complete agent action cycle: thinking -> tool calls -> response
type TurnMetaData struct {
	// Turn tracking
	TurnID       string `json:"turn_id"`
	TurnSequence int    `json:"turn_sequence"` // Position within the turn (0 = user, 1+ = assistant parts)

	// Chain of thought (collapsible in UI)
	Thinking *ThinkingBlock `json:"thinking,omitempty"`

	// Tool execution group (supports parallel calls)
	ToolGroup *ToolGroup `json:"tool_group,omitempty"`

	// Artifacts (diffs, sources, generated content)
	Artifacts []Artifact `json:"artifacts,omitempty"`

	// Context information (inherited from MessageMetaData)
	Context *MessageContext `json:"context,omitempty"`
}

// ThinkingBlock represents chain-of-thought reasoning
type ThinkingBlock struct {
	Content    string `json:"content"`
	DurationMs int64  `json:"duration_ms,omitempty"`
	Visible    bool   `json:"visible"` // Whether to show in UI by default
}

// ToolGroup represents a group of tool calls that can be executed in parallel
type ToolGroup struct {
	GroupID string           `json:"group_id"`
	Status  ToolGroupStatus  `json:"status"`
	Calls   []ToolCallRecord `json:"calls"`
}

// ToolGroupStatus represents the status of a tool group
type ToolGroupStatus string

const (
	ToolGroupStatusPending   ToolGroupStatus = "pending"
	ToolGroupStatusRunning   ToolGroupStatus = "running"
	ToolGroupStatusCompleted ToolGroupStatus = "completed"
	ToolGroupStatusError     ToolGroupStatus = "error"
)

// ToolCallRecord represents a single tool call within a group
type ToolCallRecord struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Input       map[string]interface{} `json:"input"`
	Status      ToolCallStatus         `json:"status"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	StartedAt   string                 `json:"started_at"`
	CompletedAt string                 `json:"completed_at,omitempty"`
	DurationMs  int64                  `json:"duration_ms,omitempty"`
}

// ToolCallStatus represents the status of an individual tool call
type ToolCallStatus string

const (
	ToolCallStatusPending   ToolCallStatus = "pending"
	ToolCallStatusRunning   ToolCallStatus = "running"
	ToolCallStatusCompleted ToolCallStatus = "completed"
	ToolCallStatusError     ToolCallStatus = "error"
)

// Artifact represents a UI artifact (diff, sources, answer, etc.)
type Artifact struct {
	ID     string                 `json:"id"`
	Type   ArtifactType           `json:"type"`
	Status ArtifactStatus         `json:"status"`
	Data   map[string]interface{} `json:"data"`
}

// ArtifactType represents the type of artifact
type ArtifactType string

const (
	NewArtifactTypeDiff        ArtifactType = "diff"
	NewArtifactTypeSources     ArtifactType = "sources"
	NewArtifactTypeAnswer      ArtifactType = "answer"
	NewArtifactTypeContent     ArtifactType = "content_generation"
	NewArtifactTypeImagePrompt ArtifactType = "image_prompt"
)

// ArtifactStatus represents the status of an artifact
type ArtifactStatus string

const (
	ArtifactStatusPendingNew  ArtifactStatus = "pending"
	ArtifactStatusAcceptedNew ArtifactStatus = "accepted"
	ArtifactStatusRejectedNew ArtifactStatus = "rejected"
)

// ToolCategory represents the category of a tool for UI grouping
type ToolCategory string

const (
	ToolCategoryResearch   ToolCategory = "research"
	ToolCategoryAnalysis   ToolCategory = "analysis"
	ToolCategoryEditing    ToolCategory = "editing"
	ToolCategoryGeneration ToolCategory = "generation"
)

// ToolCategoryInfo provides metadata about tool categories
var ToolCategories = map[string]ToolCategory{
	"search_web_sources":       ToolCategoryResearch,
	"ask_question":             ToolCategoryResearch,
	"get_relevant_sources":     ToolCategoryResearch,
	"fetch_url":                ToolCategoryResearch,
	"add_context_from_sources": ToolCategoryAnalysis,
	"edit_text":                ToolCategoryEditing,
	"generate_text_content":    ToolCategoryGeneration,
	"generate_image_prompt":    ToolCategoryGeneration,
}

// IsParallelizable returns whether a tool can be executed in parallel with others
func IsParallelizable(toolName string) bool {
	parallelTools := map[string]bool{
		"search_web_sources":       true,
		"ask_question":             true,
		"get_relevant_sources":     true,
		"fetch_url":                true,
		"add_context_from_sources": true,
	}
	return parallelTools[toolName]
}

// HasArtifact returns whether a tool produces an artifact
func HasArtifact(toolName string) bool {
	artifactTools := map[string]bool{
		"search_web_sources":    true,
		"ask_question":          true,
		"get_relevant_sources":  true,
		"edit_text":             true,
		"generate_text_content": true,
		"generate_image_prompt": true,
	}
	return artifactTools[toolName]
}
