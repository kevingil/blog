package repository

import (
	"context"
	"encoding/json"
	"time"

	"backend/pkg/core"
	"backend/pkg/database/models"
	"backend/pkg/types"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// DataSourceRepository provides data access for data sources
type DataSourceRepository struct {
	db *gorm.DB
}

// NewDataSourceRepository creates a new DataSourceRepository
func NewDataSourceRepository(db *gorm.DB) *DataSourceRepository {
	return &DataSourceRepository{db: db}
}

// dataSourceModelToType converts a database model to types
func dataSourceModelToType(m *models.DataSource) *types.DataSource {
	var metaData map[string]interface{}
	if m.MetaData != nil {
		_ = json.Unmarshal(m.MetaData, &metaData)
	}

	return &types.DataSource{
		ID:               m.ID,
		OrganizationID:   m.OrganizationID,
		Name:             m.Name,
		URL:              m.URL,
		FeedURL:          m.FeedURL,
		SourceType:       m.SourceType,
		CrawlFrequency:   m.CrawlFrequency,
		IsEnabled:        m.IsEnabled,
		IsDiscovered:     m.IsDiscovered,
		DiscoveredFromID: m.DiscoveredFromID,
		LastCrawledAt:    m.LastCrawledAt,
		NextCrawlAt:      m.NextCrawlAt,
		CrawlStatus:      m.CrawlStatus,
		ErrorMessage:     m.ErrorMessage,
		ContentCount:     m.ContentCount,
		MetaData:         metaData,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}

// dataSourceTypeToModel converts a types type to database model
func dataSourceTypeToModel(s *types.DataSource) *models.DataSource {
	var metaData datatypes.JSON
	if s.MetaData != nil {
		data, _ := json.Marshal(s.MetaData)
		metaData = datatypes.JSON(data)
	}

	return &models.DataSource{
		ID:               s.ID,
		OrganizationID:   s.OrganizationID,
		Name:             s.Name,
		URL:              s.URL,
		FeedURL:          s.FeedURL,
		SourceType:       s.SourceType,
		CrawlFrequency:   s.CrawlFrequency,
		IsEnabled:        s.IsEnabled,
		IsDiscovered:     s.IsDiscovered,
		DiscoveredFromID: s.DiscoveredFromID,
		LastCrawledAt:    s.LastCrawledAt,
		NextCrawlAt:      s.NextCrawlAt,
		CrawlStatus:      s.CrawlStatus,
		ErrorMessage:     s.ErrorMessage,
		ContentCount:     s.ContentCount,
		MetaData:         metaData,
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
	}
}

// FindByID retrieves a data source by its ID
func (r *DataSourceRepository) FindByID(ctx context.Context, id uuid.UUID) (*types.DataSource, error) {
	var model models.DataSource
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return dataSourceModelToType(&model), nil
}

// FindByOrganizationID retrieves all data sources for an organization
func (r *DataSourceRepository) FindByOrganizationID(ctx context.Context, orgID uuid.UUID) ([]types.DataSource, error) {
	var dataSourceModels []models.DataSource
	if err := r.db.WithContext(ctx).Where("organization_id = ?", orgID).Order("created_at DESC").Find(&dataSourceModels).Error; err != nil {
		return nil, err
	}

	sources := make([]types.DataSource, len(dataSourceModels))
	for i, m := range dataSourceModels {
		sources[i] = *dataSourceModelToType(&m)
	}
	return sources, nil
}

// FindByURL checks if a data source with the given URL exists
func (r *DataSourceRepository) FindByURL(ctx context.Context, url string) (*types.DataSource, error) {
	var model models.DataSource
	if err := r.db.WithContext(ctx).Where("url = ?", url).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return dataSourceModelToType(&model), nil
}

// FindDueToCrawl retrieves data sources that are due for crawling
func (r *DataSourceRepository) FindDueToCrawl(ctx context.Context, limit int) ([]types.DataSource, error) {
	var dataSourceModels []models.DataSource
	now := time.Now()
	if err := r.db.WithContext(ctx).
		Where("is_enabled = ? AND (next_crawl_at IS NULL OR next_crawl_at <= ?)", true, now).
		Where("crawl_status != ?", "crawling").
		Order("next_crawl_at ASC NULLS FIRST").
		Limit(limit).
		Find(&dataSourceModels).Error; err != nil {
		return nil, err
	}

	sources := make([]types.DataSource, len(dataSourceModels))
	for i, m := range dataSourceModels {
		sources[i] = *dataSourceModelToType(&m)
	}
	return sources, nil
}

// Save creates a new data source
func (r *DataSourceRepository) Save(ctx context.Context, source *types.DataSource) error {
	model := dataSourceTypeToModel(source)
	if model.ID == uuid.Nil {
		model.ID = uuid.New()
		source.ID = model.ID
	}
	return r.db.WithContext(ctx).Create(model).Error
}

// Update updates an existing data source
func (r *DataSourceRepository) Update(ctx context.Context, source *types.DataSource) error {
	model := dataSourceTypeToModel(source)
	return r.db.WithContext(ctx).Save(model).Error
}

// UpdateCrawlStatus updates the crawl status of a data source
func (r *DataSourceRepository) UpdateCrawlStatus(ctx context.Context, id uuid.UUID, status string, errorMsg *string) error {
	updates := map[string]interface{}{
		"crawl_status": status,
		"updated_at":   time.Now(),
	}
	if status == "success" {
		now := time.Now()
		updates["last_crawled_at"] = now
		updates["error_message"] = nil
	} else if errorMsg != nil {
		updates["error_message"] = *errorMsg
	}

	return r.db.WithContext(ctx).Model(&models.DataSource{}).Where("id = ?", id).Updates(updates).Error
}

// UpdateNextCrawlAt updates the next crawl time
func (r *DataSourceRepository) UpdateNextCrawlAt(ctx context.Context, id uuid.UUID, nextCrawlAt time.Time) error {
	return r.db.WithContext(ctx).Model(&models.DataSource{}).
		Where("id = ?", id).
		Update("next_crawl_at", nextCrawlAt).Error
}

// IncrementContentCount increments the content count for a data source
func (r *DataSourceRepository) IncrementContentCount(ctx context.Context, id uuid.UUID, delta int) error {
	return r.db.WithContext(ctx).Model(&models.DataSource{}).
		Where("id = ?", id).
		Update("content_count", gorm.Expr("content_count + ?", delta)).Error
}

// Delete removes a data source by its ID
func (r *DataSourceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.DataSource{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.ErrNotFound
	}
	return nil
}

// List retrieves all data sources with pagination
func (r *DataSourceRepository) List(ctx context.Context, offset, limit int) ([]types.DataSource, int64, error) {
	var dataSourceModels []models.DataSource
	var total int64

	if err := r.db.WithContext(ctx).Model(&models.DataSource{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&dataSourceModels).Error; err != nil {
		return nil, 0, err
	}

	sources := make([]types.DataSource, len(dataSourceModels))
	for i, m := range dataSourceModels {
		sources[i] = *dataSourceModelToType(&m)
	}
	return sources, total, nil
}
