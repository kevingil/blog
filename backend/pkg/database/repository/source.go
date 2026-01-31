package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"backend/pkg/core"
	"backend/pkg/core/source"
	"backend/pkg/database/models"
	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// SourceRepository provides data access for article sources
type SourceRepository struct {
	db *gorm.DB
}

// NewSourceRepository creates a new SourceRepository
func NewSourceRepository(db *gorm.DB) *SourceRepository {
	return &SourceRepository{db: db}
}

// sourceModelToType converts a database model to types
func sourceModelToType(m *models.Source) *types.Source {
	var metaData map[string]interface{}
	if m.MetaData != nil {
		_ = json.Unmarshal(m.MetaData, &metaData)
	}

	var embedding []float32
	if m.Embedding.Slice() != nil {
		embedding = m.Embedding.Slice()
	}

	return &types.Source{
		ID:         m.ID,
		ArticleID:  m.ArticleID,
		Title:      m.Title,
		Content:    m.Content,
		URL:        m.URL,
		SourceType: m.SourceType,
		Embedding:  embedding,
		MetaData:   metaData,
		CreatedAt:  m.CreatedAt,
	}
}

// sourceTypeToModel converts a types type to database model
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

// FindByID retrieves a source by its ID
func (r *SourceRepository) FindByID(ctx context.Context, id uuid.UUID) (*types.Source, error) {
	var model models.Source
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return sourceModelToType(&model), nil
}

// FindByArticleID retrieves all sources for an article
func (r *SourceRepository) FindByArticleID(ctx context.Context, articleID uuid.UUID) ([]types.Source, error) {
	var sourceModels []models.Source
	if err := r.db.WithContext(ctx).Where("article_id = ?", articleID).Order("created_at DESC").Find(&sourceModels).Error; err != nil {
		return nil, err
	}

	sources := make([]types.Source, len(sourceModels))
	for i, m := range sourceModels {
		sources[i] = *sourceModelToType(&m)
	}
	return sources, nil
}

// sourceWithArticleRow represents the joined query result
type sourceWithArticleRow struct {
	models.Source
	ArticleTitle string `gorm:"column:article_title"`
	ArticleSlug  string `gorm:"column:article_slug"`
}

// List retrieves all sources with pagination and article metadata
func (r *SourceRepository) List(ctx context.Context, opts source.SourceListOptions) ([]source.SourceWithArticle, int64, error) {
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.PerPage < 1 || opts.PerPage > 100 {
		opts.PerPage = 20
	}

	offset := (opts.Page - 1) * opts.PerPage

	var total int64
	if err := r.db.WithContext(ctx).Model(&models.Source{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []sourceWithArticleRow
	err := r.db.WithContext(ctx).Table("article_source").
		Select("article_source.*, article.draft_title as article_title, article.slug as article_slug").
		Joins("LEFT JOIN article ON article.id = article_source.article_id").
		Order("article_source.created_at DESC").
		Offset(offset).
		Limit(opts.PerPage).
		Scan(&rows).Error

	if err != nil {
		return nil, 0, err
	}

	results := make([]source.SourceWithArticle, len(rows))
	for i, row := range rows {
		results[i] = source.SourceWithArticle{
			Source:       *sourceModelToType(&row.Source),
			ArticleTitle: row.ArticleTitle,
			ArticleSlug:  row.ArticleSlug,
		}
	}

	return results, total, nil
}

// SearchSimilar performs vector similarity search for sources within an article
func (r *SourceRepository) SearchSimilar(ctx context.Context, articleID uuid.UUID, embedding []float32, limit int) ([]types.Source, error) {
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

	sources := make([]types.Source, len(sourceModels))
	for i, m := range sourceModels {
		sources[i] = *sourceModelToType(&m)
	}
	return sources, nil
}

// Save creates a new source
func (r *SourceRepository) Save(ctx context.Context, source *types.Source) error {
	model := sourceTypeToModel(source)
	if model.ID == uuid.Nil {
		model.ID = uuid.New()
		source.ID = model.ID
	}
	return r.db.WithContext(ctx).Create(model).Error
}

// Update updates an existing source
func (r *SourceRepository) Update(ctx context.Context, source *types.Source) error {
	model := sourceTypeToModel(source)
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
