package adapters

import (
	"context"
	"encoding/json"

	"backend/pkg/core/ml/llm/tools"
	coreSource "backend/pkg/core/source"
	"backend/pkg/database"
	"backend/pkg/database/models"
	"backend/pkg/database/repository"
	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"gorm.io/datatypes"
)

// SourceServiceAdapter adapts core/source Service to match the tools.ExaSourceService interface
type SourceServiceAdapter struct{}

// NewSourceServiceAdapter creates a new adapter for the source service
func NewSourceServiceAdapter() *SourceServiceAdapter {
	return &SourceServiceAdapter{}
}

// getService creates a new source service with repository injection
func (a *SourceServiceAdapter) getService() *coreSource.Service {
	db := database.DB()
	sourceRepo := repository.NewSourceRepository(db)
	articleRepo := repository.NewArticleRepository(db)
	return coreSource.NewService(sourceRepo, articleRepo)
}

// sourceTypeToModel converts types.Source to models.Source
func sourceTypeToModel(s *types.Source) *models.Source {
	var metaData datatypes.JSON
	if s.MetaData != nil {
		data, _ := json.Marshal(s.MetaData)
		metaData = datatypes.JSON(data)
	}

	var embedding pgvector.Vector
	if len(s.Embedding) > 0 {
		embedding = pgvector.NewVector(s.Embedding)
	}

	return &models.Source{
		ID:         s.ID,
		ArticleID:  s.ArticleID,
		Title:      s.Title,
		Content:    s.Content,
		URL:        s.URL,
		SourceType: s.SourceType,
		Embedding:  embedding,
		MetaData:   metaData,
		CreatedAt:  s.CreatedAt,
	}
}

// ScrapeAndCreateSource implements the tools.ExaSourceService interface
func (a *SourceServiceAdapter) ScrapeAndCreateSource(ctx context.Context, articleID uuid.UUID, targetURL string) (*models.Source, error) {
	svc := a.getService()
	source, err := svc.ScrapeAndCreate(ctx, articleID, targetURL)
	if err != nil {
		return nil, err
	}
	return sourceTypeToModel(source), nil
}

// CreateSource implements the tools.ExaSourceService interface
func (a *SourceServiceAdapter) CreateSource(ctx context.Context, req tools.CreateSourceRequest) (*models.Source, error) {
	svc := a.getService()
	// Convert tools.CreateSourceRequest to core source.CreateRequest
	serviceReq := coreSource.CreateRequest{
		ArticleID:  req.ArticleID,
		Title:      req.Title,
		Content:    req.Content,
		URL:        req.URL,
		SourceType: req.SourceType,
	}

	source, err := svc.Create(ctx, serviceReq)
	if err != nil {
		return nil, err
	}
	return sourceTypeToModel(source), nil
}
