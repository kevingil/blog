package tools

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
)

type ToolInfo struct {
	Name        string
	Description string
	Parameters  map[string]any
	Required    []string
}

type toolResponseType string

type (
	sessionIDContextKey      string
	messageIDContextKey      string
	articleIDContextKey       string
	documentStateContextKey  string
)

const (
	ToolResponseTypeText  toolResponseType = "text"
	ToolResponseTypeImage toolResponseType = "image"

	SessionIDContextKey      sessionIDContextKey      = "session_id"
	MessageIDContextKey      messageIDContextKey      = "message_id"
	ArticleIDContextKey      articleIDContextKey       = "article_id"
	DocumentStateContextKey  documentStateContextKey  = "document_state"
)

// DocumentState holds the mutable working copy of the document during an agent turn.
// Stored as a pointer in context so both read_document and edit_text share the same state.
// After edit_text produces new content, it updates this state so subsequent read_document
// calls return the latest version (solving the stale-read problem for multi-edit turns).
type DocumentState struct {
	mu       sync.RWMutex
	HTML     string
	Markdown string
}

// DraftSaver is the interface that edit_text uses to persist draft content to the DB
// after each successful edit. This makes the backend the source of truth for draft content.
type DraftSaver interface {
	UpdateDraftContent(ctx context.Context, articleID string, htmlContent string) error
}

type ToolResponse struct {
	Type     toolResponseType `json:"type"`
	Content  string           `json:"content"`
	Metadata string           `json:"metadata,omitempty"`
	IsError  bool             `json:"is_error"`
	// NEW: Structured result for UI rendering
	Result map[string]interface{} `json:"result,omitempty"`
	// NEW: Artifact to create (if any)
	Artifact *ArtifactHint `json:"artifact,omitempty"`
}

// ArtifactHint provides hints for creating UI artifacts from tool results
type ArtifactHint struct {
	Type string                 `json:"type"` // "diff", "sources", "answer", "content_generation", "image_prompt"
	Data map[string]interface{} `json:"data"`
}

// ArtifactHintType constants
const (
	ArtifactHintTypeDiff        = "diff"
	ArtifactHintTypeSources     = "sources"
	ArtifactHintTypeAnswer      = "answer"
	ArtifactHintTypeContent     = "content_generation"
	ArtifactHintTypeImagePrompt = "image_prompt"
)

func NewTextResponse(content string) ToolResponse {
	return ToolResponse{
		Type:    ToolResponseTypeText,
		Content: content,
	}
}

// NewTextResponseWithResult creates a response with structured result data
func NewTextResponseWithResult(content string, result map[string]interface{}) ToolResponse {
	return ToolResponse{
		Type:    ToolResponseTypeText,
		Content: content,
		Result:  result,
	}
}

// NewTextResponseWithArtifact creates a response with an artifact hint
func NewTextResponseWithArtifact(content string, result map[string]interface{}, artifactType string, artifactData map[string]interface{}) ToolResponse {
	return ToolResponse{
		Type:    ToolResponseTypeText,
		Content: content,
		Result:  result,
		Artifact: &ArtifactHint{
			Type: artifactType,
			Data: artifactData,
		},
	}
}

func WithResponseMetadata(response ToolResponse, metadata any) ToolResponse {
	if metadata != nil {
		metadataBytes, err := json.Marshal(metadata)
		if err != nil {
			return response
		}
		response.Metadata = string(metadataBytes)
	}
	return response
}

func NewTextErrorResponse(content string) ToolResponse {
	return ToolResponse{
		Type:    ToolResponseTypeText,
		Content: content,
		IsError: true,
	}
}

type ToolCall struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Input string `json:"input"`
}

type BaseTool interface {
	Info() ToolInfo
	Run(ctx context.Context, params ToolCall) (ToolResponse, error)
}

func GetContextValues(ctx context.Context) (string, string) {
	sessionID := ctx.Value(SessionIDContextKey)
	messageID := ctx.Value(MessageIDContextKey)
	if sessionID == nil {
		return "", ""
	}
	if messageID == nil {
		return sessionID.(string), ""
	}
	return sessionID.(string), messageID.(string)
}

func GetArticleIDFromContext(ctx context.Context) string {
	articleID := ctx.Value(ArticleIDContextKey)
	if articleID == nil {
		return ""
	}
	return articleID.(string)
}

// WithArticleID adds article ID to context for tools that need it
func WithArticleID(ctx context.Context, articleID string) context.Context {
	return context.WithValue(ctx, ArticleIDContextKey, articleID)
}

// WithDocumentContent creates a mutable DocumentState and stores a pointer in context.
// Both read_document and edit_text operate on this shared state so the agent always
// sees the latest content during multi-edit turns.
// The markdown is unescaped before storing so the LLM sees clean content that it can reproduce.
func WithDocumentContent(ctx context.Context, html, markdown string) context.Context {
	if markdown != "" {
		markdown = unescapeMarkdown(markdown)
	}
	state := &DocumentState{HTML: html, Markdown: markdown}
	return context.WithValue(ctx, DocumentStateContextKey, state)
}

// getDocumentState retrieves the mutable DocumentState pointer from context.
func getDocumentState(ctx context.Context) *DocumentState {
	state := ctx.Value(DocumentStateContextKey)
	if state == nil {
		return nil
	}
	return state.(*DocumentState)
}

// GetDocumentHTMLFromContext retrieves the current HTML document content from context.
func GetDocumentHTMLFromContext(ctx context.Context) string {
	state := getDocumentState(ctx)
	if state == nil {
		return ""
	}
	state.mu.RLock()
	defer state.mu.RUnlock()
	return state.HTML
}

// GetDocumentMarkdownFromContext retrieves the current markdown document content from context.
func GetDocumentMarkdownFromContext(ctx context.Context) string {
	state := getDocumentState(ctx)
	if state == nil {
		return ""
	}
	state.mu.RLock()
	defer state.mu.RUnlock()
	return state.Markdown
}

// UpdateDocumentMarkdown updates the in-memory document state after an edit.
// This ensures subsequent read_document calls return the post-edit content.
func UpdateDocumentMarkdown(ctx context.Context, newMarkdown string) {
	state := getDocumentState(ctx)
	if state == nil {
		return
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	state.Markdown = newMarkdown
}

// unescapeMarkdown removes Turndown's backslash escapes that make LLM matching impossible.
func unescapeMarkdown(s string) string {
	r := strings.NewReplacer(
		`\*`, `*`,
		`\_`, `_`,
		`\[`, `[`,
		`\]`, `]`,
		`\#`, `#`,
		`\>`, `>`,
		`\-`, `-`,
		`\+`, `+`,
		`\~`, `~`,
		`\|`, `|`,
	)
	// Also unescape backticks: \` -> `
	result := r.Replace(s)
	result = strings.ReplaceAll(result, "\\`", "`")
	return result
}
