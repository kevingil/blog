package adapters

import (
	"backend/pkg/api/source"
	"backend/pkg/core/ml/llm/tools"
	"backend/pkg/database/models"
	"context"

	"github.com/google/uuid"
)

// SourceServiceAdapter adapts source.ArticleSourceService to match the tools.ExaSourceService interface
type SourceServiceAdapter struct {
	service *source.ArticleSourceService
}

// NewSourceServiceAdapter creates a new adapter for the ArticleSourceService
func NewSourceServiceAdapter(service *source.ArticleSourceService) *SourceServiceAdapter {
	return &SourceServiceAdapter{
		service: service,
	}
}

// ScrapeAndCreateSource implements the tools.ExaSourceService interface
func (a *SourceServiceAdapter) ScrapeAndCreateSource(ctx context.Context, articleID uuid.UUID, targetURL string) (*models.Source, error) {
	return a.service.ScrapeAndCreateSource(ctx, articleID, targetURL)
}

// CreateSource implements the tools.ExaSourceService interface
func (a *SourceServiceAdapter) CreateSource(ctx context.Context, req tools.CreateSourceRequest) (*models.Source, error) {
	// Convert tools.CreateSourceRequest to source.CreateSourceRequest
	serviceReq := source.CreateSourceRequest{
		ArticleID:  req.ArticleID,
		Title:      req.Title,
		Content:    req.Content,
		URL:        req.URL,
		SourceType: req.SourceType,
	}

	return a.service.CreateSource(ctx, serviceReq)
}
