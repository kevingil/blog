package adapters

import (
	"context"

	"backend/pkg/core/ml/llm/tools"
	"backend/pkg/integrations/exa"
)

// ExaServiceAdapter adapts exa.Client to match the tools.ExaSearchService interface
type ExaServiceAdapter struct {
	client *exa.Client
}

// NewExaServiceAdapter creates a new adapter for the Exa client
func NewExaServiceAdapter(client *exa.Client) *ExaServiceAdapter {
	return &ExaServiceAdapter{
		client: client,
	}
}

// SearchWithDefaults implements the tools.ExaSearchService interface
func (a *ExaServiceAdapter) SearchWithDefaults(ctx context.Context, query string) (*tools.ExaSearchResponse, error) {
	resp, err := a.client.SearchWithDefaults(ctx, query)
	if err != nil {
		return nil, err
	}

	// Convert client response to tools response
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
	return a.client.IsConfigured()
}

// AnswerWithDefaults implements the tools.ExaAnswerService interface
func (a *ExaServiceAdapter) AnswerWithDefaults(ctx context.Context, question string) (*tools.ExaAnswerResponse, error) {
	resp, err := a.client.AnswerWithDefaults(ctx, question)
	if err != nil {
		return nil, err
	}

	// Convert client response to tools response
	toolsResp := &tools.ExaAnswerResponse{
		Answer:      resp.Answer,
		CostDollars: resp.CostDollars,
	}

	// Convert citations
	for _, citation := range resp.Citations {
		toolsCitation := tools.ExaAnswerCitation{
			ID:            citation.ID,
			URL:           citation.URL,
			Title:         citation.Title,
			Author:        citation.Author,
			PublishedDate: citation.PublishedDate,
			Text:          citation.Text,
			Image:         citation.Image,
			Favicon:       citation.Favicon,
			Extras:        citation.Extras,
		}
		toolsResp.Citations = append(toolsResp.Citations, toolsCitation)
	}

	return toolsResp, nil
}
