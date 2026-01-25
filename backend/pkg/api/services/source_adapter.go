package services

import (
	"backend/pkg/core/ml/llm/tools"
	"backend/pkg/models"
	"context"

	"github.com/google/uuid"
)

// SourceServiceAdapter adapts ArticleSourceService to match the tools.ExaSourceService interface
type SourceServiceAdapter struct {
	service *ArticleSourceService
}

// NewSourceServiceAdapter creates a new adapter for the ArticleSourceService
func NewSourceServiceAdapter(service *ArticleSourceService) *SourceServiceAdapter {
	return &SourceServiceAdapter{
		service: service,
	}
}

// ScrapeAndCreateSource implements the tools.ExaSourceService interface
func (a *SourceServiceAdapter) ScrapeAndCreateSource(ctx context.Context, articleID uuid.UUID, targetURL string) (*models.ArticleSource, error) {
	return a.service.ScrapeAndCreateSource(ctx, articleID, targetURL)
}

// CreateSource implements the tools.ExaSourceService interface
func (a *SourceServiceAdapter) CreateSource(ctx context.Context, req tools.CreateSourceRequest) (*models.ArticleSource, error) {
	// Convert tools.CreateSourceRequest to services.CreateSourceRequest
	serviceReq := CreateSourceRequest{
		ArticleID:  req.ArticleID,
		Title:      req.Title,
		Content:    req.Content,
		URL:        req.URL,
		SourceType: req.SourceType,
	}

	return a.service.CreateSource(ctx, serviceReq)
}
