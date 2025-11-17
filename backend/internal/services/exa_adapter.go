package services

import (
	"context"

	"blog-agent-go/backend/internal/core/ml/llm/tools"
)

// ExaServiceAdapter adapts ExaSearchService to match the tools.ExaSearchService interface
type ExaServiceAdapter struct {
	service *ExaSearchService
}

// NewExaServiceAdapter creates a new adapter for the Exa service
func NewExaServiceAdapter(service *ExaSearchService) *ExaServiceAdapter {
	return &ExaServiceAdapter{
		service: service,
	}
}

// SearchWithDefaults implements the tools.ExaSearchService interface
func (a *ExaServiceAdapter) SearchWithDefaults(ctx context.Context, query string) (*tools.ExaSearchResponse, error) {
	resp, err := a.service.SearchWithDefaults(ctx, query)
	if err != nil {
		return nil, err
	}

	// Convert service response to tools response
	toolsResp := &tools.ExaSearchResponse{
		RequestID:          resp.RequestID,
		ResolvedSearchType: resp.ResolvedSearchType,
		SearchType:         resp.SearchType,
		Context:            resp.Context,
		CostDollars:        resp.CostDollars,
	}

	// Convert results
	for _, result := range resp.Results {
		toolsResult := tools.ExaSearchResult{
			Title:         result.Title,
			URL:           result.URL,
			ID:            result.ID,
			PublishedDate: result.PublishedDate,
			Author:        result.Author,
			Text:          result.Text,
			Highlights:    result.Highlights,
			Summary:       result.Summary,
			Image:         result.Image,
			Favicon:       result.Favicon,
			Score:         result.Score,
			Extras:        result.Extras,
		}
		toolsResp.Results = append(toolsResp.Results, toolsResult)
	}

	return toolsResp, nil
}

// IsConfigured implements the tools.ExaSearchService interface
func (a *ExaServiceAdapter) IsConfigured() bool {
	return a.service.IsConfigured()
}
