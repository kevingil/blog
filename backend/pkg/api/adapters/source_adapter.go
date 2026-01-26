package adapters

import (
	"backend/pkg/core/ml/llm/tools"
	coreSource "backend/pkg/core/source"
	"backend/pkg/database/models"
	"context"

	"github.com/google/uuid"
)

// SourceServiceAdapter adapts core/source functions to match the tools.ExaSourceService interface
type SourceServiceAdapter struct{}

// NewSourceServiceAdapter creates a new adapter for the source service
func NewSourceServiceAdapter() *SourceServiceAdapter {
	return &SourceServiceAdapter{}
}

// ScrapeAndCreateSource implements the tools.ExaSourceService interface
func (a *SourceServiceAdapter) ScrapeAndCreateSource(ctx context.Context, articleID uuid.UUID, targetURL string) (*models.Source, error) {
	return coreSource.ScrapeAndCreate(ctx, articleID, targetURL)
}

// CreateSource implements the tools.ExaSourceService interface
func (a *SourceServiceAdapter) CreateSource(ctx context.Context, req tools.CreateSourceRequest) (*models.Source, error) {
	// Convert tools.CreateSourceRequest to core source.CreateRequest
	serviceReq := coreSource.CreateRequest{
		ArticleID:  req.ArticleID,
		Title:      req.Title,
		Content:    req.Content,
		URL:        req.URL,
		SourceType: req.SourceType,
	}

	return coreSource.Create(ctx, serviceReq)
}
