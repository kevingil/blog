package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
)

// ExaAnswerService interface for Exa answer operations
type ExaAnswerService interface {
	AnswerWithDefaults(ctx context.Context, question string) (*ExaAnswerResponse, error)
	IsConfigured() bool
}

// ExaAnswerResponse represents the response from Exa answer API
type ExaAnswerResponse struct {
	Answer      string                 `json:"answer"`
	Citations   []ExaAnswerCitation    `json:"citations"`
	CostDollars map[string]interface{} `json:"costDollars,omitempty"`
}

// ExaAnswerCitation represents a citation from the Exa answer API
type ExaAnswerCitation struct {
	ID            string                 `json:"id"`
	URL           string                 `json:"url"`
	Title         string                 `json:"title"`
	Author        string                 `json:"author,omitempty"`
	PublishedDate string                 `json:"publishedDate,omitempty"`
	Text          string                 `json:"text,omitempty"`
	Image         string                 `json:"image,omitempty"`
	Favicon       string                 `json:"favicon,omitempty"`
	Extras        map[string]interface{} `json:"extras,omitempty"`
}

// ExaAnswerTool gets direct answers to questions using Exa's /answer endpoint
type ExaAnswerTool struct {
	exaService ExaAnswerService
}

// NewExaAnswerTool creates a new Exa Answer tool
func NewExaAnswerTool(exaService ExaAnswerService) *ExaAnswerTool {
	return &ExaAnswerTool{
		exaService: exaService,
	}
}

func (t *ExaAnswerTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "ask_question",
		Description: "Get a direct, factual answer to a specific question with citations. Use this for factual queries like 'What is X?', 'How does Y work?', 'When did Z happen?'. Returns a concise answer with source citations. For exploratory research or finding multiple sources, use search_web_sources instead.",
		Parameters: map[string]any{
			"question": map[string]any{
				"type":        "string",
				"description": "A specific question to answer. Be clear and specific for best results. Examples: 'What is the current market cap of Apple?', 'How does photosynthesis work?', 'When was the Eiffel Tower built?'",
			},
		},
		Required: []string{"question"},
	}
}

func (t *ExaAnswerTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
	var input struct {
		Question string `json:"question"`
	}

	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		log.Printf("❓ [ExaAnswer] ERROR: Failed to parse input: %v", err)
		return NewTextErrorResponse("Invalid input format"), err
	}

	if input.Question == "" {
		log.Printf("❓ [ExaAnswer] ERROR: Empty question provided")
		return NewTextErrorResponse("question is required"), fmt.Errorf("question is required")
	}

	log.Printf("❓ [ExaAnswer] Asking question: %q", input.Question)

	// Check if service is configured
	if t.exaService == nil || !t.exaService.IsConfigured() {
		log.Printf("❓ [ExaAnswer] ERROR: Exa service not configured")
		return NewTextErrorResponse("Exa answer service is not configured. Please set the EXA_API_KEY environment variable."), fmt.Errorf("Exa service not configured")
	}

	// Get answer from Exa
	answerResp, err := t.exaService.AnswerWithDefaults(ctx, input.Question)
	if err != nil {
		log.Printf("❓ [ExaAnswer] ERROR: Failed to get answer: %v", err)
		return NewTextErrorResponse(fmt.Sprintf("Failed to get answer: %v", err)), err
	}

	log.Printf("❓ [ExaAnswer] ✅ Received answer with %d citations", len(answerResp.Citations))

	// Build citations list for result
	var citations []map[string]interface{}
	for _, citation := range answerResp.Citations {
		citationData := map[string]interface{}{
			"url":   citation.URL,
			"title": citation.Title,
		}
		if citation.Author != "" {
			citationData["author"] = citation.Author
		}
		if citation.PublishedDate != "" {
			citationData["published_date"] = citation.PublishedDate
		}
		if citation.Favicon != "" {
			citationData["favicon"] = citation.Favicon
		}
		if citation.Text != "" {
			// Truncate text for preview
			textPreview := citation.Text
			if len(textPreview) > 300 {
				textPreview = textPreview[:300] + "..."
			}
			citationData["text_preview"] = textPreview
		}
		citations = append(citations, citationData)
	}

	// Prepare result
	result := map[string]interface{}{
		"answer":         answerResp.Answer,
		"citations":      citations,
		"question":       input.Question,
		"citation_count": len(citations),
		"tool_name":      "ask_question",
	}

	// Add cost information if available
	if answerResp.CostDollars != nil {
		result["cost_info"] = answerResp.CostDollars
	}

	log.Printf("❓ [ExaAnswer] Answer preview: %s...", truncateString(answerResp.Answer, 100))

	// Create artifact hint for answer display
	artifactData := map[string]interface{}{
		"answer":    answerResp.Answer,
		"citations": citations,
		"question":  input.Question,
	}

	resultJSON, _ := json.Marshal(result)
	return ToolResponse{
		Type:    ToolResponseTypeText,
		Content: string(resultJSON),
		Result:  result,
		Artifact: &ArtifactHint{
			Type: ArtifactHintTypeAnswer,
			Data: artifactData,
		},
	}, nil
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
