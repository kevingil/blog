// Package metadata provides structured types for message metadata
package metadata

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// NewToolExecution creates a new tool execution record
func NewToolExecution(toolName, toolID string, input, output interface{}, duration time.Duration, err error) *ToolExecution {
	te := &ToolExecution{
		ToolName:   toolName,
		ToolID:     toolID,
		Input:      input,
		Output:     output,
		Duration:   duration.Milliseconds(),
		ExecutedAt: time.Now(),
		Success:    err == nil,
	}

	if err != nil {
		te.Error = err.Error()
	}

	return te
}

// NewMessageContext creates a new message context
func NewMessageContext(articleID, sessionID, requestID, userID string) *MessageContext {
	return &MessageContext{
		ArticleID: articleID,
		SessionID: sessionID,
		RequestID: requestID,
		UserID:    userID,
	}
}

// WithDocumentHash adds document hash to context
func (c *MessageContext) WithDocumentHash(content string) *MessageContext {
	hash := sha256.Sum256([]byte(content))
	c.DocumentHash = hex.EncodeToString(hash[:])
	return c
}

// NewUserAction creates a new user action record
func NewUserAction(action, artifactID, feedback, reason string) *UserAction {
	return &UserAction{
		Action:     action,
		Timestamp:  time.Now(),
		ArtifactID: artifactID,
		Feedback:   feedback,
		Reason:     reason,
	}
}

// BuildMetaData creates a complete metadata structure
func BuildMetaData() *MessageMetaData {
	return &MessageMetaData{}
}

// WithArtifact adds artifact info to metadata
func (m *MessageMetaData) WithArtifact(artifact *ArtifactInfo) *MessageMetaData {
	m.Artifact = artifact
	return m
}

// WithToolExecution adds tool execution to metadata
func (m *MessageMetaData) WithToolExecution(tool *ToolExecution) *MessageMetaData {
	m.ToolExecution = tool
	return m
}

// WithContext adds context to metadata
func (m *MessageMetaData) WithContext(context *MessageContext) *MessageMetaData {
	m.Context = context
	return m
}

// WithUserAction adds user action to metadata
func (m *MessageMetaData) WithUserAction(action *UserAction) *MessageMetaData {
	m.UserAction = action
	return m
}

// WithThinking adds thinking/reasoning content to metadata (LEGACY - use WithSteps for new code)
func (m *MessageMetaData) WithThinking(thinking *ThinkingBlock) *MessageMetaData {
	m.Thinking = thinking
	return m
}

// WithSteps adds chain of thought steps to metadata
func (m *MessageMetaData) WithSteps(steps []ChainOfThoughtStep) *MessageMetaData {
	m.Steps = steps
	return m
}

// Build returns the final metadata structure (for explicit termination of builder chain)
func (m *MessageMetaData) Build() *MessageMetaData {
	return m
}
