package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"backend/pkg/core"
	"backend/pkg/database/models"
	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// CrawledContentRepository provides data access for crawled content
type CrawledContentRepository struct {
	db *gorm.DB
}

// NewCrawledContentRepository creates a new CrawledContentRepository
func NewCrawledContentRepository(db *gorm.DB) *CrawledContentRepository {
	return &CrawledContentRepository{db: db}
}

// crawledContentModelToType converts a database model to types
func crawledContentModelToType(m *models.CrawledContent) *types.CrawledContent {
	var metaData map[string]interface{}
	if m.MetaData != nil {
		_ = json.Unmarshal(m.MetaData, &metaData)
	}

	var embedding []float32
	if m.Embedding.Slice() != nil {
		embedding = m.Embedding.Slice()
	}

	return &types.CrawledContent{
		ID:           m.ID,
		DataSourceID: m.DataSourceID,
		URL:          m.URL,
		Title:        m.Title,
		Content:      m.Content,
		Summary:      m.Summary,
		Author:       m.Author,
		PublishedAt:  m.PublishedAt,
		Embedding:    embedding,
		MetaData:     metaData,
		CreatedAt:    m.CreatedAt,
	}
}

// crawledContentTypeToModel converts a types type to database model
func crawledContentTypeToModel(c *types.CrawledContent) *models.CrawledContent {
	var metaData datatypes.JSON
	if c.MetaData != nil {
		data, _ := json.Marshal(c.MetaData)
		metaData = datatypes.JSON(data)
	}

	var embedding pgvector.Vector
	if len(c.Embedding) > 0 {
		embedding = pgvector.NewVector(c.Embedding)
	}

	return &models.CrawledContent{
		ID:           c.ID,
		DataSourceID: c.DataSourceID,
		URL:          c.URL,
		Title:        c.Title,
		Content:      c.Content,
		Summary:      c.Summary,
		Author:       c.Author,
		PublishedAt:  c.PublishedAt,
		Embedding:    embedding,
		MetaData:     metaData,
		CreatedAt:    c.CreatedAt,
	}
}

// FindByID retrieves crawled content by its ID
func (r *CrawledContentRepository) FindByID(ctx context.Context, id uuid.UUID) (*types.CrawledContent, error) {
	var model models.CrawledContent
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return crawledContentModelToType(&model), nil
}

// FindByDataSourceID retrieves all crawled content for a data source
func (r *CrawledContentRepository) FindByDataSourceID(ctx context.Context, dataSourceID uuid.UUID, offset, limit int) ([]types.CrawledContent, int64, error) {
	var contentModels []models.CrawledContent
	var total int64

	if err := r.db.WithContext(ctx).Model(&models.CrawledContent{}).Where("data_source_id = ?", dataSourceID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).
		Where("data_source_id = ?", dataSourceID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&contentModels).Error; err != nil {
		return nil, 0, err
	}

	contents := make([]types.CrawledContent, len(contentModels))
	for i, m := range contentModels {
		contents[i] = *crawledContentModelToType(&m)
	}
	return contents, total, nil
}

// FindByURL checks if content with the given URL exists for a data source
func (r *CrawledContentRepository) FindByURL(ctx context.Context, dataSourceID uuid.UUID, url string) (*types.CrawledContent, error) {
	var model models.CrawledContent
	if err := r.db.WithContext(ctx).Where("data_source_id = ? AND url = ?", dataSourceID, url).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return crawledContentModelToType(&model), nil
}

// FindByIDs retrieves multiple crawled content by their IDs
func (r *CrawledContentRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]types.CrawledContent, error) {
	var contentModels []models.CrawledContent
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&contentModels).Error; err != nil {
		return nil, err
	}

	contents := make([]types.CrawledContent, len(contentModels))
	for i, m := range contentModels {
		contents[i] = *crawledContentModelToType(&m)
	}
	return contents, nil
}

// SearchSimilar performs vector similarity search for crawled content
func (r *CrawledContentRepository) SearchSimilar(ctx context.Context, embedding []float32, limit int) ([]types.CrawledContent, error) {
	var contentModels []models.CrawledContent

	embeddingVector := pgvector.NewVector(embedding)
	query := fmt.Sprintf(
		"SELECT * FROM crawled_content WHERE embedding IS NOT NULL ORDER BY embedding <=> '%s' LIMIT %d",
		embeddingVector.String(),
		limit,
	)

	if err := r.db.WithContext(ctx).Raw(query).Scan(&contentModels).Error; err != nil {
		return nil, err
	}

	contents := make([]types.CrawledContent, len(contentModels))
	for i, m := range contentModels {
		contents[i] = *crawledContentModelToType(&m)
	}
	return contents, nil
}

// SearchSimilarByOrg performs vector similarity search for crawled content within an organization
func (r *CrawledContentRepository) SearchSimilarByOrg(ctx context.Context, orgID uuid.UUID, embedding []float32, limit int) ([]types.CrawledContent, error) {
	var contentModels []models.CrawledContent

	embeddingVector := pgvector.NewVector(embedding)
	query := fmt.Sprintf(
		`SELECT cc.* FROM crawled_content cc 
		 JOIN data_source ds ON cc.data_source_id = ds.id 
		 WHERE ds.organization_id = '%s' AND cc.embedding IS NOT NULL 
		 ORDER BY cc.embedding <=> '%s' LIMIT %d`,
		orgID.String(),
		embeddingVector.String(),
		limit,
	)

	if err := r.db.WithContext(ctx).Raw(query).Scan(&contentModels).Error; err != nil {
		return nil, err
	}

	contents := make([]types.CrawledContent, len(contentModels))
	for i, m := range contentModels {
		contents[i] = *crawledContentModelToType(&m)
	}
	return contents, nil
}

// Save creates new crawled content (upserts based on data_source_id + url)
func (r *CrawledContentRepository) Save(ctx context.Context, content *types.CrawledContent) error {
	model := crawledContentTypeToModel(content)
	if model.ID == uuid.Nil {
		model.ID = uuid.New()
		content.ID = model.ID
	}

	// Upsert: update if exists, create if not
	return r.db.WithContext(ctx).
		Where("data_source_id = ? AND url = ?", model.DataSourceID, model.URL).
		Assign(model).
		FirstOrCreate(model).Error
}

// Update updates existing crawled content
func (r *CrawledContentRepository) Update(ctx context.Context, content *types.CrawledContent) error {
	model := crawledContentTypeToModel(content)
	return r.db.WithContext(ctx).Save(model).Error
}

// Delete removes crawled content by its ID
func (r *CrawledContentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.CrawledContent{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.ErrNotFound
	}
	return nil
}

// DeleteByDataSourceID removes all crawled content for a data source
func (r *CrawledContentRepository) DeleteByDataSourceID(ctx context.Context, dataSourceID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("data_source_id = ?", dataSourceID).Delete(&models.CrawledContent{}).Error
}

// CountByDataSourceID counts crawled content for a data source
func (r *CrawledContentRepository) CountByDataSourceID(ctx context.Context, dataSourceID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.CrawledContent{}).Where("data_source_id = ?", dataSourceID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// FindRecentByOrg retrieves recent crawled content for an organization
func (r *CrawledContentRepository) FindRecentByOrg(ctx context.Context, orgID uuid.UUID, limit int) ([]types.CrawledContent, error) {
	var contentModels []models.CrawledContent

	if err := r.db.WithContext(ctx).
		Joins("JOIN data_source ds ON crawled_content.data_source_id = ds.id").
		Where("ds.organization_id = ?", orgID).
		Order("crawled_content.created_at DESC").
		Limit(limit).
		Find(&contentModels).Error; err != nil {
		return nil, err
	}

	contents := make([]types.CrawledContent, len(contentModels))
	for i, m := range contentModels {
		contents[i] = *crawledContentModelToType(&m)
	}
	return contents, nil
}
