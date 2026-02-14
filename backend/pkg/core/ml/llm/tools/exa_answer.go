package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"backend/pkg/integrations/exa"
)

// ExaAnswerService interface for Exa answer operations
// Satisfied directly by exa.Client
type ExaAnswerService interface {
	AnswerWithDefaults(ctx context.Context, question string) (*exa.AnswerResponse, error)
	IsConfigured() bool
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
		Description: "Ask a specific factual question and get an answer with citations. Use this LIBERALLY -- ask many questions to build comprehensive context before writing. Ask follow-up questions based on earlier answers. Be highly specific: name people, technologies, timeframes, and metrics. Good: 'What performance improvements did Shopify report after migrating to HTMX in 2024?' Bad: 'What are the benefits of HTMX?'",
		Parameters: map[string]any{
			"question": map[string]any{
				"type":        "string",
				"description": "A specific, detailed question. Include names, dates, metrics, or technologies. Examples: 'What did Carson Gross say about HTMX 2.0 adoption at GopherCon 2024?', 'What is the median TTFB for Go+HTMX apps vs React SPAs in 2024 benchmarks?'",
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
