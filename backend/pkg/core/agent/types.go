// Package agent provides types and infrastructure for the agent system
package agent

// ChatMessage is a simplified representation of a chat message from the frontend.
// It intentionally mirrors the OpenAI message schema but without advanced fields (tool calls, etc.)
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is what the /agent endpoint receives from the frontend.
// It contains a single user message, document context, and the article ID for loading conversation history.
type ChatRequest struct {
	Message         string `json:"message" validate:"required,min=1"`
	DocumentContent string `json:"documentContent,omitempty"`
	ArticleID       string `json:"articleId" validate:"required"` // Required for loading context
}

// ChatRequestResponse is the immediate response returned when a chat request is submitted
type ChatRequestResponse struct {
	RequestID string `json:"requestId"`
	Status    string `json:"status"`
}

// StreamResponse represents streaming events sent to the client via WebSocket.
// Supports block-based streaming for structured agent responses.
//
// Supported block types:
// - "content_delta": Real-time content chunks as they're generated
// - "text": Complete assistant text responses (for backward compatibility)
// - "tool_use": Tool/function calls made by the agent (legacy)
// - "tool_result": Results returned from tool executions (legacy)
// - "tool_group_start": Start of a tool group (new architecture)
// - "tool_status": Status update for an individual tool in a group
// - "tool_group_complete": All tools in group have completed
// - "full_message": Complete message with meta_data for artifact tools
// - "artifact": Artifact created from tool results
// - "user": User messages (streamed as initial context)
// - "system": System messages (streamed as initial context)
// - "thinking": Thinking/chain-of-thought content
// - "error": Error messages
// - "done": Completion signal
type StreamResponse struct {
	RequestID string `json:"requestId,omitempty"`
	Type      string `json:"type"`
	Content   string `json:"content,omitempty"`
	Iteration int    `json:"iteration,omitempty"`

	// Tool-specific fields for tool_use blocks (legacy)
	ToolID    string      `json:"tool_id,omitempty"`
	ToolName  string      `json:"tool_name,omitempty"`
	ToolInput interface{} `json:"tool_input,omitempty"`

	// Tool result fields for tool_result blocks (legacy)
	ToolResult interface{} `json:"tool_result,omitempty"`

	// NEW: Tool group for parallel tool execution
	ToolGroup *ToolGroupPayload `json:"tool_group,omitempty"`

	// NEW: Individual tool status update within a group
	ToolStatus *ToolStatusPayload `json:"tool_status,omitempty"`

	// NEW: Artifact created from tool results
	Artifact *ArtifactPayload `json:"artifact,omitempty"`

	// NEW: Chain of thought content
	ThinkingContent string `json:"thinking_content,omitempty"`

	// Full message for artifact tools (edit_text, rewrite_document, search_web_sources)
	// Contains complete meta_data structure matching database format
	FullMessage *FullMessagePayload `json:"full_message,omitempty"`

	// Thinking-specific fields (legacy - use ThinkingContent instead)
	ThinkingMessage string `json:"thinking_message,omitempty"`

	// Legacy fields for backward compatibility
	Role  string `json:"role,omitempty"`
	Data  any    `json:"data,omitempty"`
	Done  bool   `json:"done,omitempty"`
	Error string `json:"error,omitempty"`
}

// Stream event type constants
const (
	StreamTypeContentDelta      = "content_delta"
	StreamTypeReasoningDelta    = "reasoning_delta"
	StreamTypeText              = "text"
	StreamTypeToolUse           = "tool_use"
	StreamTypeToolResult        = "tool_result"
	StreamTypeToolGroupStart    = "tool_group_start"
	StreamTypeToolStatus        = "tool_status"
	StreamTypeToolGroupComplete = "tool_group_complete"
	StreamTypeFullMessage       = "full_message"
	StreamTypeArtifact          = "artifact"
	StreamTypeUser              = "user"
	StreamTypeSystem            = "system"
	StreamTypeThinking          = "thinking"
	StreamTypeError             = "error"
	StreamTypeDone              = "done"
)

// ToolGroupPayload represents a group of tool calls for streaming
type ToolGroupPayload struct {
	GroupID string            `json:"group_id"`
	Status  string            `json:"status"` // "pending", "running", "completed", "error"
	Calls   []ToolCallPayload `json:"calls"`
}

// ToolCallPayload represents a single tool call in a group
type ToolCallPayload struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Input       map[string]interface{} `json:"input,omitempty"`
	Status      string                 `json:"status"` // "pending", "running", "completed", "error"
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	StartedAt   string                 `json:"started_at,omitempty"`
	CompletedAt string                 `json:"completed_at,omitempty"`
	DurationMs  int64                  `json:"duration_ms,omitempty"`
}

// ToolStatusPayload represents a status update for an individual tool
type ToolStatusPayload struct {
	GroupID     string                 `json:"group_id"`
	ToolID      string                 `json:"tool_id"`
	Name        string                 `json:"name"`
	Status      string                 `json:"status"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	CompletedAt string                 `json:"completed_at,omitempty"`
	DurationMs  int64                  `json:"duration_ms,omitempty"`
}

// ArtifactPayload represents an artifact for streaming
type ArtifactPayload struct {
	ID     string                 `json:"id"`
	Type   string                 `json:"type"`   // "diff", "sources", "answer", "content_generation", "image_prompt"
	Status string                 `json:"status"` // "pending", "accepted", "rejected"
	Data   map[string]interface{} `json:"data"`
}

// FullMessagePayload is the complete message structure matching what's saved to DB
// Used to stream full messages for artifact tools so frontend can render immediately
type FullMessagePayload struct {
	ID        string                 `json:"id"`
	ArticleID string                 `json:"article_id"`
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	MetaData  map[string]interface{} `json:"meta_data"`
	CreatedAt string                 `json:"created_at"`
}

// ArtifactUpdate represents tool execution status shown to user
type ArtifactUpdate struct {
	ToolName string      `json:"tool_name"`
	Status   string      `json:"status"` // "starting", "in_progress", "completed", "error"
	Message  string      `json:"message"`
	Result   interface{} `json:"result,omitempty"`
	Error    string      `json:"error,omitempty"`
}
