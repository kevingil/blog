// Package metadata provides structured types for message metadata
package metadata

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

// NewArtifactInfo creates a new artifact info structure
func NewArtifactInfo(artifactType, content, title, description string) *ArtifactInfo {
	return &ArtifactInfo{
		ID:          uuid.New().String(),
		Type:        artifactType,
		Status:      ArtifactStatusPending,
		Content:     content,
		Title:       title,
		Description: description,
	}
}

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

// NewTaskStatus creates a new task status record
func NewTaskStatus(taskID, name string) *TaskStatus {
	return &TaskStatus{
		TaskID:    taskID,
		Name:      name,
		Status:    TaskStatusQueued,
		Progress:  0,
		StartedAt: time.Now(),
	}
}

// Start marks a task as in progress
func (t *TaskStatus) Start() {
	t.Status = TaskStatusInProgress
	t.StartedAt = time.Now()
}

// Complete marks a task as completed
func (t *TaskStatus) Complete() {
	t.Status = TaskStatusCompleted
	t.Progress = 100
	now := time.Now()
	t.CompletedAt = &now
}

// Fail marks a task as failed
func (t *TaskStatus) Fail(err error) {
	t.Status = TaskStatusFailed
	now := time.Now()
	t.CompletedAt = &now
	if err != nil {
		t.Error = err.Error()
	}
}

// UpdateProgress updates task progress
func (t *TaskStatus) UpdateProgress(progress float64) {
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	t.Progress = progress
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

// NewAttachment creates a new attachment
func NewAttachment(attachmentType, name, url, mimeType string, size int64) Attachment {
	return Attachment{
		Type:     attachmentType,
		Name:     name,
		URL:      url,
		MimeType: mimeType,
		Size:     size,
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

// WithTaskStatus adds task status to metadata
func (m *MessageMetaData) WithTaskStatus(task *TaskStatus) *MessageMetaData {
	m.TaskStatus = task
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

// WithAttachments adds attachments to metadata
func (m *MessageMetaData) WithAttachments(attachments []Attachment) *MessageMetaData {
	m.Attachments = attachments
	return m
}

// WithThinking adds thinking/reasoning content to metadata
func (m *MessageMetaData) WithThinking(thinking *ThinkingBlock) *MessageMetaData {
	m.Thinking = thinking
	return m
}

// Build returns the final metadata structure (for explicit termination of builder chain)
func (m *MessageMetaData) Build() *MessageMetaData {
	return m
}

