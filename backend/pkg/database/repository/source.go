package repository

import (
	"context"
	"fmt"

	"backend/pkg/core"
	"backend/pkg/core/source"
	"backend/pkg/database/models"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
)

// SourceRepository implements source.SourceStore using GORM
type SourceRepository struct {
	db *gorm.DB
}

// NewSourceRepository creates a new SourceRepository
func NewSourceRepository(db *gorm.DB) *SourceRepository {
	return &SourceRepository{db: db}
}

// FindByID retrieves a source by its ID
func (r *SourceRepository) FindByID(ctx context.Context, id uuid.UUID) (*source.Source, error) {
	var model models.Source
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return model.ToCore(), nil
}

// FindByArticleID retrieves all sources for an article
func (r *SourceRepository) FindByArticleID(ctx context.Context, articleID uuid.UUID) ([]source.Source, error) {
	var sourceModels []models.Source
	if err := r.db.WithContext(ctx).Where("article_id = ?", articleID).Order("created_at DESC").Find(&sourceModels).Error; err != nil {
		return nil, err
	}

	sources := make([]source.Source, len(sourceModels))
	for i, m := range sourceModels {
		sources[i] = *m.ToCore()
	}

	return sources, nil
}

// SearchSimilar performs vector similarity search for sources within an article
func (r *SourceRepository) SearchSimilar(ctx context.Context, articleID uuid.UUID, embedding []float32, limit int) ([]source.Source, error) {
	var sourceModels []models.Source

	embeddingVector := pgvector.NewVector(embedding)
	query := fmt.Sprintf(
		"SELECT * FROM article_source WHERE article_id = '%s' AND embedding IS NOT NULL ORDER BY embedding <-> '%s' LIMIT %d",
		articleID.String(),
		embeddingVector.String(),
		limit,
	)

	if err := r.db.WithContext(ctx).Raw(query).Scan(&sourceModels).Error; err != nil {
		return nil, err
	}

	sources := make([]source.Source, len(sourceModels))
	for i, m := range sourceModels {
		sources[i] = *m.ToCore()
	}

	return sources, nil
}

// Save creates a new source
func (r *SourceRepository) Save(ctx context.Context, s *source.Source) error {
	model := models.SourceFromCore(s)

	if s.ID == uuid.Nil {
		s.ID = uuid.New()
		model.ID = s.ID
	}

	return r.db.WithContext(ctx).Create(model).Error
}

// Update updates an existing source
func (r *SourceRepository) Update(ctx context.Context, s *source.Source) error {
	model := models.SourceFromCore(s)
	return r.db.WithContext(ctx).Save(model).Error
}

// Delete removes a source by its ID
func (r *SourceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Source{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.ErrNotFound
	}
	return nil
}
