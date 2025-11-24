// Package agent provides types and infrastructure for the agent system
package agent

// ChatMessage is a simplified representation of a chat message from the frontend.
// It intentionally mirrors the OpenAI message schema but without advanced fields (tool calls, etc.)
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is what the /agent endpoint receives from the frontend.
// It contains the full chat transcript, optional model selection, and document context.
type ChatRequest struct {
	Messages        []ChatMessage `json:"messages" validate:"required,min=1"`
	Model           string        `json:"model"`
	DocumentContent string        `json:"documentContent,omitempty"`
	ArticleID       string        `json:"articleId,omitempty"`
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
// - "text": Assistant text responses
// - "tool_use": Tool/function calls made by the agent
// - "tool_result": Results returned from tool executions
// - "user": User messages (streamed as initial context)
// - "system": System messages (streamed as initial context)
// - "thinking": Thinking state during tool execution
// - "error": Error messages
// - "done": Completion signal
type StreamResponse struct {
	RequestID string `json:"requestId,omitempty"`
	Type      string `json:"type"` // "text", "tool_use", "tool_result", "thinking", "error", "done"
	Content   string `json:"content,omitempty"`
	Iteration int    `json:"iteration,omitempty"`

	// Tool-specific fields for tool_use blocks
	ToolID    string      `json:"tool_id,omitempty"`
	ToolName  string      `json:"tool_name,omitempty"`
	ToolInput interface{} `json:"tool_input,omitempty"`

	// Tool result fields for tool_result blocks
	ToolResult interface{} `json:"tool_result,omitempty"`

	// Thinking-specific fields
	ThinkingMessage string `json:"thinking_message,omitempty"`

	// Legacy fields for backward compatibility
	Role  string `json:"role,omitempty"`
	Data  any    `json:"data,omitempty"`
	Done  bool   `json:"done,omitempty"`
	Error string `json:"error,omitempty"`
}

// ArtifactUpdate represents tool execution status shown to user
type ArtifactUpdate struct {
	ToolName string      `json:"tool_name"`
	Status   string      `json:"status"` // "starting", "in_progress", "completed", "error"
	Message  string      `json:"message"`
	Result   interface{} `json:"result,omitempty"`
	Error    string      `json:"error,omitempty"`
}
