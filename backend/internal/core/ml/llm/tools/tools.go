package tools

import (
	"context"
	"encoding/json"
)

type ToolInfo struct {
	Name        string
	Description string
	Parameters  map[string]any
	Required    []string
}

type toolResponseType string

type (
	sessionIDContextKey        string
	messageIDContextKey        string
	articleIDContextKey        string
	documentContentContextKey  string
	documentMarkdownContextKey string
)

const (
	ToolResponseTypeText  toolResponseType = "text"
	ToolResponseTypeImage toolResponseType = "image"

	SessionIDContextKey        sessionIDContextKey        = "session_id"
	MessageIDContextKey        messageIDContextKey        = "message_id"
	ArticleIDContextKey        articleIDContextKey        = "article_id"
	DocumentContentContextKey  documentContentContextKey  = "document_content"
	DocumentMarkdownContextKey documentMarkdownContextKey = "document_markdown"
)

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

// WithDocumentContent adds document content (both HTML and markdown) to context for tools
func WithDocumentContent(ctx context.Context, html, markdown string) context.Context {
	ctx = context.WithValue(ctx, DocumentContentContextKey, html)
	ctx = context.WithValue(ctx, DocumentMarkdownContextKey, markdown)
	return ctx
}

// GetDocumentHTMLFromContext retrieves the original HTML document content from context
func GetDocumentHTMLFromContext(ctx context.Context) string {
	html := ctx.Value(DocumentContentContextKey)
	if html == nil {
		return ""
	}
	return html.(string)
}

// GetDocumentMarkdownFromContext retrieves the markdown version of the document from context
func GetDocumentMarkdownFromContext(ctx context.Context) string {
	markdown := ctx.Value(DocumentMarkdownContextKey)
	if markdown == nil {
		return ""
	}
	return markdown.(string)
}
