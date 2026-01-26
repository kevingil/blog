package message

import (
	"encoding/base64"
	"fmt"
	"time"
)

type Role string

const (
	User      Role = "user"
	Assistant Role = "assistant"
	System    Role = "system"
	Tool      Role = "tool"
)

type FinishReason string

const (
	FinishReasonEndTurn          FinishReason = "end_turn"
	FinishReasonToolUse          FinishReason = "tool_use"
	FinishReasonCanceled         FinishReason = "canceled"
	FinishReasonPermissionDenied FinishReason = "permission_denied"
	FinishReasonUnknown          FinishReason = "unknown"
	FinishReasonMaxTokens        FinishReason = "max_tokens"
)

type ContentPart interface {
	isContentPart()
}

type TextContent struct {
	Text string `json:"text"`
}

func (t TextContent) isContentPart() {}

type BinaryContent struct {
	Path     string `json:"path"`
	MIMEType string `json:"mime_type"`
	Data     []byte `json:"data"`
}

func (b BinaryContent) isContentPart() {}

// String returns a string representation of binary content for the given provider
func (b BinaryContent) String(providerType string) string {
	// For now, return the data URL format - you may need to customize this
	// based on the specific provider requirements
	return fmt.Sprintf("data:%s;base64,%s", b.MIMEType, base64.StdEncoding.EncodeToString(b.Data))
}

type ToolCall struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Input    string `json:"input"`
	Type     string `json:"type,omitempty"`
	Finished bool   `json:"finished,omitempty"`
}

// ToolCall implements ContentPart so it can be stored in Parts for persistence
func (t ToolCall) isContentPart() {}

type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
	Metadata   string `json:"metadata,omitempty"`
	IsError    bool   `json:"is_error"`
}

func (t ToolResult) isContentPart() {}

type Finish struct {
	Reason FinishReason `json:"reason"`
	Time   int64        `json:"time"`
}

func (f Finish) isContentPart() {}

type Attachment struct {
	FilePath string `json:"file_path"`
	MimeType string `json:"mime_type"`
	Content  []byte `json:"content"`
}

type Message struct {
	ID          string        `json:"id"`
	SessionID   string        `json:"session_id"`
	Role        Role          `json:"role"`
	Parts       []ContentPart `json:"parts"`
	Model       string        `json:"model,omitempty"`
	CreatedAt   int64         `json:"created_at"`
	UpdatedAt   int64         `json:"updated_at"`
	finishParts []Finish      `json:"-"`
}

func (m *Message) AddToolCall(call ToolCall) {
	m.Parts = append(m.Parts, call)
}

func (m *Message) ToolCalls() []ToolCall {
	var calls []ToolCall
	for _, part := range m.Parts {
		if tc, ok := part.(ToolCall); ok {
			calls = append(calls, tc)
		}
	}
	return calls
}

func (m *Message) SetToolCalls(calls []ToolCall) {
	// Remove existing tool calls from Parts
	var newParts []ContentPart
	for _, part := range m.Parts {
		if _, ok := part.(ToolCall); !ok {
			newParts = append(newParts, part)
		}
	}
	m.Parts = newParts
	// Add new tool calls
	for _, call := range calls {
		m.Parts = append(m.Parts, call)
	}
}

func (m *Message) FinishToolCall(id string) {
	// Mark tool call as finished - implementation depends on your needs
}

func (m *Message) AppendToolCallInput(id, input string) {
	// Update tool call input - implementation depends on your needs
}

func (m *Message) AppendContent(content string) {
	for i, part := range m.Parts {
		if textPart, ok := part.(TextContent); ok {
			m.Parts[i] = TextContent{Text: textPart.Text + content}
			return
		}
	}
	// If no text part exists, create one
	m.Parts = append(m.Parts, TextContent{Text: content})
}

func (m *Message) AddFinish(reason FinishReason) {
	finish := Finish{
		Reason: reason,
		Time:   time.Now().Unix(),
	}
	m.finishParts = append(m.finishParts, finish)
	m.Parts = append(m.Parts, finish)
}

func (m *Message) FinishReason() FinishReason {
	if len(m.finishParts) == 0 {
		return ""
	}
	return m.finishParts[len(m.finishParts)-1].Reason
}

// Content returns a helper for accessing text content
func (m *Message) Content() ContentHelper {
	return ContentHelper{message: m}
}

// BinaryContent returns all binary content parts
func (m *Message) BinaryContent() []BinaryContent {
	var binaryParts []BinaryContent
	for _, part := range m.Parts {
		if binaryPart, ok := part.(BinaryContent); ok {
			binaryParts = append(binaryParts, binaryPart)
		}
	}
	return binaryParts
}

// ContentHelper provides methods for accessing message content
type ContentHelper struct {
	message *Message
}

// String returns the concatenated text content
func (c ContentHelper) String() string {
	var content string
	for _, part := range c.message.Parts {
		if textPart, ok := part.(TextContent); ok {
			content += textPart.Text
		}
	}
	return content
}

// ToolResults returns all tool results from the message
func (m *Message) ToolResults() []ToolResult {
	var results []ToolResult
	for _, part := range m.Parts {
		if toolResult, ok := part.(ToolResult); ok {
			results = append(results, toolResult)
		}
	}
	return results
}

type CreateMessageParams struct {
	Role  Role          `json:"role"`
	Parts []ContentPart `json:"parts"`
	Model string        `json:"model,omitempty"`
}
