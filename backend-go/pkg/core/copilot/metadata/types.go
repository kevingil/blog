// Package metadata provides structured types for message metadata
package metadata

import "time"

// MessageMetaData represents the complete metadata structure for a chat message
type MessageMetaData struct {
	// Artifact information - for content suggestions/changes
	Artifact *ArtifactInfo `json:"artifact,omitempty"`

	// Tool execution details
	ToolExecution *ToolExecution `json:"tool_execution,omitempty"`

	// Context information
	Context *MessageContext `json:"context,omitempty"`

	// User actions (accept/reject)
	UserAction *UserAction `json:"user_action,omitempty"`

	// Chain of thought reasoning (from reasoning models) - LEGACY, use Steps instead
	Thinking *ThinkingBlock `json:"thinking,omitempty"`

	// Chain of thought steps (reasoning -> tool -> reasoning -> content)
	Steps []ChainOfThoughtStep `json:"steps,omitempty"`
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

// ChainOfThoughtStep represents a single step in the chain of thought
// Steps are ordered: reasoning -> tool -> reasoning -> content
type ChainOfThoughtStep struct {
	Type      string         `json:"type"`                // "reasoning", "tool", "content"
	Reasoning *ThinkingBlock `json:"reasoning,omitempty"` // Present when Type == "reasoning"
	Tool      *ToolStepInfo  `json:"tool,omitempty"`      // Present when Type == "tool"
	Content   string         `json:"content,omitempty"`   // Present when Type == "content"
}

// ToolStepInfo represents tool execution info in a chain of thought step
type ToolStepInfo struct {
	ToolID      string                 `json:"tool_id"`
	ToolName    string                 `json:"tool_name"`
	Input       map[string]interface{} `json:"input,omitempty"`
	Output      map[string]interface{} `json:"output,omitempty"`
	Status      string                 `json:"status"` // "running", "completed", "error"
	Error       string                 `json:"error,omitempty"`
	StartedAt   string                 `json:"started_at,omitempty"`
	CompletedAt string                 `json:"completed_at,omitempty"`
	DurationMs  int64                  `json:"duration_ms,omitempty"`
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
