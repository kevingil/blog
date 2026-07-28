package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	coreSource "backend/pkg/core/source"
	"backend/pkg/types"

	"github.com/google/uuid"
)

type SelectSourcesForEditTool struct {
	sourceService ArticleSourceService
}

func NewSelectSourcesForEditTool(sourceService ArticleSourceService) *SelectSourcesForEditTool {
	return &SelectSourcesForEditTool{sourceService: sourceService}
}

func (t *SelectSourcesForEditTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "select_sources_for_edit",
		Description: "Persist selected sources for the pending edit and return the exact excerpts to use as edit context. Use this after research or get_relevant_sources and before replace_lines.",
		Parameters: map[string]any{
			"sources": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"source_id": map[string]any{
							"type":        []string{"string", "null"},
							"description": "Existing article source ID when selecting a stored source.",
						},
						"title": map[string]any{
							"type":        []string{"string", "null"},
							"description": "Source title. Required for new sources created from ask_question citations.",
						},
						"url": map[string]any{
							"type":        []string{"string", "null"},
							"description": "Source URL.",
						},
						"source_type": map[string]any{
							"type":        []string{"string", "null"},
							"description": "Source type such as web or web_search.",
						},
						"excerpt_text": map[string]any{
							"type":        "string",
							"description": "The exact source excerpt to keep in edit context.",
						},
						"excerpt_id": map[string]any{
							"type":        []string{"string", "null"},
							"description": "Stable excerpt identifier returned by get_relevant_sources when available.",
						},
						"content": map[string]any{
							"type":        []string{"string", "null"},
							"description": "Optional full source content. For ask_question citations, this can match excerpt_text.",
						},
						"origin_tool": map[string]any{
							"type":        []string{"string", "null"},
							"description": "Where this source came from, such as ask_question or search_web_sources.",
						},
						"origin_query": map[string]any{
							"type":        []string{"string", "null"},
							"description": "Original search query when applicable.",
						},
						"origin_question": map[string]any{
							"type":        []string{"string", "null"},
							"description": "Original research question when applicable.",
						},
						"author": map[string]any{
							"type":        []string{"string", "null"},
							"description": "Source author when known.",
						},
						"published_date": map[string]any{
							"type":        []string{"string", "null"},
							"description": "Publication date when known.",
						},
					},
					"required": []string{"excerpt_text"},
				},
				"description": "Sources to persist and include in edit context.",
			},
		},
		Required: []string{"sources"},
	}
}

func (t *SelectSourcesForEditTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
	var input struct {
		Sources []struct {
			SourceID       string `json:"source_id"`
			Title          string `json:"title"`
			URL            string `json:"url"`
			SourceType     string `json:"source_type"`
			ExcerptText    string `json:"excerpt_text"`
			ExcerptID      string `json:"excerpt_id"`
			Content        string `json:"content"`
			OriginTool     string `json:"origin_tool"`
			OriginQuery    string `json:"origin_query"`
			OriginQuestion string `json:"origin_question"`
			Author         string `json:"author"`
			PublishedDate  string `json:"published_date"`
		} `json:"sources"`
	}

	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return NewTextErrorResponse("Invalid input format"), err
	}
	if len(input.Sources) == 0 {
		return NewTextErrorResponse("sources is required"), fmt.Errorf("sources is required")
	}

	articleIDStr := GetArticleIDFromContext(ctx)
	if articleIDStr == "" {
		return NewTextErrorResponse("No article ID available"), fmt.Errorf("article_id is required")
	}
	articleID, err := uuid.Parse(articleIDStr)
	if err != nil {
		return NewTextErrorResponse("Invalid article ID"), fmt.Errorf("invalid article id: %w", err)
	}

	requestID := GetRequestIDFromContext(ctx)
	selected := make([]SourceContextResource, 0, len(input.Sources))

	for _, src := range input.Sources {
		if src.ExcerptText == "" {
			continue
		}

		var sourceID *uuid.UUID
		if src.SourceID != "" {
			parsed, err := uuid.Parse(src.SourceID)
			if err != nil {
				return NewTextErrorResponse("Invalid source_id"), fmt.Errorf("invalid source_id %q: %w", src.SourceID, err)
			}
			sourceID = &parsed
		}

		saved, err := t.sourceService.UpsertAgentResource(ctx, coreSource.AgentResourceSelection{
			ArticleID:         articleID,
			SourceID:          sourceID,
			Title:             src.Title,
			Content:           src.Content,
			URL:               src.URL,
			SourceType:        src.SourceType,
			OriginTool:        src.OriginTool,
			OriginQuery:       src.OriginQuery,
			OriginQuestion:    src.OriginQuestion,
			Author:            src.Author,
			PublishedDate:     src.PublishedDate,
			SelectedExcerpt:   src.ExcerptText,
			SelectedExcerptID: src.ExcerptID,
			RequestID:         requestID,
			UsageStatus:       "used",
		})
		if err != nil {
			log.Printf("[SelectSourcesForEdit] failed to persist source selection: %v", err)
			return NewTextErrorResponse(fmt.Sprintf("Failed to persist source selection: %v", err)), err
		}

		resource := BuildSourceContextResources([]types.Source{*saved})
		if len(resource) > 0 {
			selected = append(selected, resource[0])
		}
	}

	inventorySources, err := t.sourceService.GetByArticleID(ctx, articleID)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("Failed to load source inventory: %v", err)), err
	}
	inventory := BuildSourceContextResources(inventorySources)

	result := map[string]interface{}{
		"selected_sources":       selected,
		"selected_count":         len(selected),
		"source_inventory":       inventory,
		"source_inventory_count": len(inventory),
		"selected_context":       FormatSelectedSourcesContext(selected),
		"inventory_context":      FormatSourceInventoryContext(inventory),
		"tool_name":              "select_sources_for_edit",
	}

	resultJSON, _ := json.Marshal(result)
	return ToolResponse{
		Type:    ToolResponseTypeText,
		Content: string(resultJSON),
		Result:  result,
		Artifact: &ArtifactHint{
			Type: ArtifactHintTypeSources,
			Data: map[string]interface{}{
				"selected_sources": selected,
				"source_inventory": inventory,
			},
		},
	}, nil
}
