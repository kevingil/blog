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
// - "tool_use": Tool/function calls made by the agent
// - "tool_result": Results returned from tool executions
// - "full_message": Complete message with meta_data for artifact tools (edit_text, rewrite_document)
// - "user": User messages (streamed as initial context)
// - "system": System messages (streamed as initial context)
// - "thinking": Thinking state during tool execution
// - "error": Error messages
// - "done": Completion signal
type StreamResponse struct {
	RequestID string `json:"requestId,omitempty"`
	Type      string `json:"type"` // "content_delta", "text", "tool_use", "tool_result", "full_message", "thinking", "error", "done"
	Content   string `json:"content,omitempty"`
	Iteration int    `json:"iteration,omitempty"`

	// Tool-specific fields for tool_use blocks
	ToolID    string      `json:"tool_id,omitempty"`
	ToolName  string      `json:"tool_name,omitempty"`
	ToolInput interface{} `json:"tool_input,omitempty"`

	// Tool result fields for tool_result blocks
	ToolResult interface{} `json:"tool_result,omitempty"`

	// Full message for artifact tools (edit_text, rewrite_document, search_web_sources)
	// Contains complete meta_data structure matching database format
	FullMessage *FullMessagePayload `json:"full_message,omitempty"`

	// Thinking-specific fields
	ThinkingMessage string `json:"thinking_message,omitempty"`

	// Legacy fields for backward compatibility
	Role  string `json:"role,omitempty"`
	Data  any    `json:"data,omitempty"`
	Done  bool   `json:"done,omitempty"`
	Error string `json:"error,omitempty"`
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
