package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// RewriteDocumentTool rewrites the entire document
type RewriteDocumentTool struct {
	textGenService TextGenerationService
}

type TextGenerationService interface {
	GenerateImagePrompt(ctx context.Context, content string) (string, error)
}

func NewRewriteDocumentTool(textGenService TextGenerationService) *RewriteDocumentTool {
	return &RewriteDocumentTool{
		textGenService: textGenService,
	}
}

func (t *RewriteDocumentTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "rewrite_document",
		Description: "Completely rewrite or significantly edit the document content",
		Parameters: map[string]any{
			"new_content": map[string]any{
				"type":        "string",
				"description": "The new document content in markdown format",
			},
			"reason": map[string]any{
				"type":        "string",
				"description": "Brief explanation of the changes made",
			},
		},
		Required: []string{"new_content", "reason"},
	}
}

func (t *RewriteDocumentTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
	var input struct {
		NewContent string `json:"new_content"`
		Reason     string `json:"reason"`
	}

	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return NewTextErrorResponse("Invalid input format"), err
	}

	if input.NewContent == "" {
		return NewTextErrorResponse("new_content is required"), fmt.Errorf("new_content is required")
	}

	result := map[string]interface{}{
		"new_content": input.NewContent,
		"reason":      input.Reason,
		"tool_name":   "rewrite_document",
	}

	resultJSON, _ := json.Marshal(result)
	return NewTextResponse(string(resultJSON)), nil
}

// EditTextTool edits specific text in the document
type EditTextTool struct{}

func NewEditTextTool() *EditTextTool {
	return &EditTextTool{}
}

func (t *EditTextTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "edit_text",
		Description: "Edit specific text in the document while preserving the rest. Use this for targeted edits, improvements, or changes to specific sections.",
		Parameters: map[string]any{
			"original_text": map[string]any{
				"type":        "string",
				"description": "The exact text to find and replace in the document",
			},
			"new_text": map[string]any{
				"type":        "string",
				"description": "The new text to replace the original text with",
			},
			"reason": map[string]any{
				"type":        "string",
				"description": "Brief explanation of why this edit is being made",
			},
		},
		Required: []string{"original_text", "new_text", "reason"},
	}
}

func (t *EditTextTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
	var input struct {
		OriginalText string `json:"original_text"`
		NewText      string `json:"new_text"`
		Reason       string `json:"reason"`
	}

	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return NewTextErrorResponse("Invalid input format"), err
	}

	if input.OriginalText == "" || input.NewText == "" {
		return NewTextErrorResponse("original_text and new_text are required"), fmt.Errorf("original_text and new_text are required")
	}

	// Generate unified diff patch using diffmatchpatch
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(input.OriginalText, input.NewText, false)
	patch := dmp.PatchMake(input.OriginalText, diffs)
	patchText := dmp.PatchToText(patch)

	// Prepare the result with patch information
	result := map[string]interface{}{
		"original_text": input.OriginalText,
		"new_text":      input.NewText,
		"reason":        input.Reason,
		"edit_type":     "patch",
		"tool_name":     "edit_text",
		"patch": map[string]interface{}{
			"unified_diff": patchText,
			"diffs":        diffs,
			"summary": map[string]interface{}{
				"additions": countDiffType(diffs, diffmatchpatch.DiffInsert),
				"deletions": countDiffType(diffs, diffmatchpatch.DiffDelete),
				"unchanged": countDiffType(diffs, diffmatchpatch.DiffEqual),
			},
		},
	}

	resultJSON, _ := json.Marshal(result)
	return NewTextResponse(string(resultJSON)), nil
}

// Helper function to count characters by diff type
func countDiffType(diffs []diffmatchpatch.Diff, diffType diffmatchpatch.Operation) int {
	count := 0
	for _, diff := range diffs {
		if diff.Type == diffType {
			count += len(diff.Text)
		}
	}
	return count
}

// AnalyzeDocumentTool analyzes document and provides suggestions
type AnalyzeDocumentTool struct{}

func NewAnalyzeDocumentTool() *AnalyzeDocumentTool {
	return &AnalyzeDocumentTool{}
}

func (t *AnalyzeDocumentTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "analyze_document",
		Description: "Analyze document and provide improvement suggestions. Can focus on specific areas or provide general analysis.",
		Parameters: map[string]any{
			"focus_area": map[string]any{
				"type":        "string",
				"description": "Optional: What aspect to focus on (structure, clarity, engagement, grammar, flow, technical_accuracy). If not provided, will analyze overall document quality.",
			},
			"user_request": map[string]any{
				"type":        "string",
				"description": "The user's original request to help understand what they want to improve",
			},
		},
		Required: []string{"user_request"},
	}
}

func (t *AnalyzeDocumentTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
	var input struct {
		FocusArea   string `json:"focus_area"`
		UserRequest string `json:"user_request"`
	}

	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return NewTextErrorResponse("Invalid input format"), err
	}

	if input.UserRequest == "" {
		return NewTextErrorResponse("user_request is required"), fmt.Errorf("user_request is required")
	}

	// Infer focus area from user request if not provided
	if input.FocusArea == "" {
		userRequestLower := strings.ToLower(input.UserRequest)
		if strings.Contains(userRequestLower, "engaging") || strings.Contains(userRequestLower, "boring") {
			input.FocusArea = "engagement"
		} else if strings.Contains(userRequestLower, "clear") || strings.Contains(userRequestLower, "confusing") {
			input.FocusArea = "clarity"
		} else if strings.Contains(userRequestLower, "structure") || strings.Contains(userRequestLower, "organize") {
			input.FocusArea = "structure"
		} else if strings.Contains(userRequestLower, "grammar") || strings.Contains(userRequestLower, "spelling") {
			input.FocusArea = "grammar"
		} else {
			input.FocusArea = "overall"
		}
	}

	result := map[string]interface{}{
		"focus_area":    input.FocusArea,
		"user_request":  input.UserRequest,
		"analysis_done": true,
		"tool_name":     "analyze_document",
	}

	resultJSON, _ := json.Marshal(result)
	return NewTextResponse(string(resultJSON)), nil
}

// GenerateImagePromptTool generates image prompts from content
type GenerateImagePromptTool struct {
	textGenService TextGenerationService
}

func NewGenerateImagePromptTool(textGenService TextGenerationService) *GenerateImagePromptTool {
	return &GenerateImagePromptTool{
		textGenService: textGenService,
	}
}

func (t *GenerateImagePromptTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "generate_image_prompt",
		Description: "Generate an image prompt based on document content",
		Parameters: map[string]any{
			"content": map[string]any{
				"type":        "string",
				"description": "The document content to generate image prompt for",
			},
		},
		Required: []string{"content"},
	}
}

func (t *GenerateImagePromptTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
	var input struct {
		Content string `json:"content"`
	}

	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return NewTextErrorResponse("Invalid input format"), err
	}

	if input.Content == "" {
		return NewTextErrorResponse("content is required"), fmt.Errorf("content is required")
	}

	prompt, err := t.textGenService.GenerateImagePrompt(ctx, input.Content)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("Failed to generate image prompt: %v", err)), err
	}

	result := map[string]interface{}{
		"prompt":    prompt,
		"tool_name": "generate_image_prompt",
	}

	resultJSON, _ := json.Marshal(result)
	return NewTextResponse(string(resultJSON)), nil
}
