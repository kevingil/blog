package datasource

import (
	"context"
	"time"

	"backend/pkg/api/dto"
	"backend/pkg/core"
	"backend/pkg/types"

	"github.com/google/uuid"
)

// Service provides business logic for data sources
type Service struct {
	dataSourceStore     DataSourceStore
	crawledContentStore CrawledContentStore
}

// NewService creates a new data source service with the provided stores
func NewService(dataSourceStore DataSourceStore, crawledContentStore CrawledContentStore) *Service {
	return &Service{
		dataSourceStore:     dataSourceStore,
		crawledContentStore: crawledContentStore,
	}
}

// GetByID retrieves a data source by its ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*dto.DataSourceResponse, error) {
	ds, err := s.dataSourceStore.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toResponse(ds), nil
}

// List retrieves all data sources for an organization
func (s *Service) List(ctx context.Context, orgID uuid.UUID) ([]dto.DataSourceResponse, error) {
	sources, err := s.dataSourceStore.FindByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	result := make([]dto.DataSourceResponse, len(sources))
	for i, ds := range sources {
		result[i] = *toResponse(&ds)
	}
	return result, nil
}

// ListByUserID retrieves all data sources for a user (without organization)
func (s *Service) ListByUserID(ctx context.Context, userID uuid.UUID) ([]dto.DataSourceResponse, error) {
	sources, err := s.dataSourceStore.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]dto.DataSourceResponse, len(sources))
	for i, ds := range sources {
		result[i] = *toResponse(&ds)
	}
	return result, nil
}

// ListAll retrieves all data sources with pagination
func (s *Service) ListAll(ctx context.Context, page, limit int) ([]dto.DataSourceResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	sources, total, err := s.dataSourceStore.List(ctx, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	result := make([]dto.DataSourceResponse, len(sources))
	for i, ds := range sources {
		result[i] = *toResponse(&ds)
	}
	return result, total, nil
}

// Create creates a new data source
// Either orgID or userID must be provided
func (s *Service) Create(ctx context.Context, orgID *uuid.UUID, userID *uuid.UUID, req dto.DataSourceCreateRequest) (*dto.DataSourceResponse, error) {
	// Validate that at least one owner is provided
	if orgID == nil && userID == nil {
		return nil, core.InvalidInputError("Either organization_id or user_id must be provided")
	}

	// Check if URL already exists
	existing, err := s.dataSourceStore.FindByURL(ctx, req.URL)
	if err != nil && err != core.ErrNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, core.ErrAlreadyExists
	}

	// Set defaults
	sourceType := req.SourceType
	if sourceType == "" {
		sourceType = "blog"
	}

	crawlFrequency := req.CrawlFrequency
	if crawlFrequency == "" {
		crawlFrequency = "daily"
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	// Calculate next crawl time
	nextCrawlAt := calculateNextCrawlTime(crawlFrequency)

	ds := &types.DataSource{
		ID:              uuid.New(),
		OrganizationID:  orgID,
		UserID:          userID,
		Name:            req.Name,
		URL:             req.URL,
		FeedURL:         req.FeedURL,
		SourceType:      sourceType,
		CrawlFrequency:  crawlFrequency,
		IsEnabled:       isEnabled,
		IsDiscovered:    false,
		CrawlStatus:     "pending",
		ContentCount:    0,
		SubscriberCount: 1,
		NextCrawlAt:     &nextCrawlAt,
	}

	if err := s.dataSourceStore.Save(ctx, ds); err != nil {
		return nil, err
	}

	return toResponse(ds), nil
}

// Update updates an existing data source
func (s *Service) Update(ctx context.Context, id uuid.UUID, req dto.DataSourceUpdateRequest) (*dto.DataSourceResponse, error) {
	ds, err := s.dataSourceStore.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if new URL already exists
	if req.URL != nil && *req.URL != ds.URL {
		existing, err := s.dataSourceStore.FindByURL(ctx, *req.URL)
		if err != nil && err != core.ErrNotFound {
			return nil, err
		}
		if existing != nil && existing.ID != id {
			return nil, core.ErrAlreadyExists
		}
		ds.URL = *req.URL
	}

	// Apply updates
	if req.Name != nil {
		ds.Name = *req.Name
	}
	if req.FeedURL != nil {
		ds.FeedURL = req.FeedURL
	}
	if req.SourceType != nil {
		ds.SourceType = *req.SourceType
	}
	if req.CrawlFrequency != nil {
		ds.CrawlFrequency = *req.CrawlFrequency
		nextCrawlAt := calculateNextCrawlTime(*req.CrawlFrequency)
		ds.NextCrawlAt = &nextCrawlAt
	}
	if req.IsEnabled != nil {
		ds.IsEnabled = *req.IsEnabled
	}

	if err := s.dataSourceStore.Update(ctx, ds); err != nil {
		return nil, err
	}

	return toResponse(ds), nil
}

// Delete removes a data source by its ID
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.dataSourceStore.Delete(ctx, id)
}

// TriggerCrawl triggers a manual crawl for a data source
func (s *Service) TriggerCrawl(ctx context.Context, id uuid.UUID) error {
	ds, err := s.dataSourceStore.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Set status to pending and clear next crawl time to prioritize it
	now := time.Now()
	ds.CrawlStatus = "pending"
	ds.NextCrawlAt = &now

	return s.dataSourceStore.Update(ctx, ds)
}

// GetContent retrieves crawled content for a data source
func (s *Service) GetContent(ctx context.Context, dataSourceID uuid.UUID, page, limit int) ([]dto.CrawledContentResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	contents, total, err := s.crawledContentStore.FindByDataSourceID(ctx, dataSourceID, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	result := make([]dto.CrawledContentResponse, len(contents))
	for i, c := range contents {
		result[i] = toContentResponse(&c)
	}
	return result, total, nil
}

// GetDueToCrawl retrieves data sources that are due for crawling
func (s *Service) GetDueToCrawl(ctx context.Context, limit int) ([]types.DataSource, error) {
	return s.dataSourceStore.FindDueToCrawl(ctx, limit)
}

// UpdateCrawlStatus updates the crawl status of a data source
func (s *Service) UpdateCrawlStatus(ctx context.Context, id uuid.UUID, status string, errorMsg *string) error {
	return s.dataSourceStore.UpdateCrawlStatus(ctx, id, status, errorMsg)
}

// SetNextCrawlTime sets the next crawl time for a data source
func (s *Service) SetNextCrawlTime(ctx context.Context, id uuid.UUID, frequency string) error {
	nextCrawlAt := calculateNextCrawlTime(frequency)
	return s.dataSourceStore.UpdateNextCrawlAt(ctx, id, nextCrawlAt)
}

// CreateDiscoveredSource creates a data source that was discovered automatically
// Either orgID or userID must be provided
func (s *Service) CreateDiscoveredSource(ctx context.Context, orgID *uuid.UUID, userID *uuid.UUID, discoveredFromID uuid.UUID, name, url string) (*dto.DataSourceResponse, error) {
	// Validate that at least one owner is provided
	if orgID == nil && userID == nil {
		return nil, core.InvalidInputError("Either organization_id or user_id must be provided")
	}

	// Check if URL already exists
	existing, err := s.dataSourceStore.FindByURL(ctx, url)
	if err != nil && err != core.ErrNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, core.ErrAlreadyExists
	}

	nextCrawlAt := calculateNextCrawlTime("daily")

	ds := &types.DataSource{
		ID:               uuid.New(),
		OrganizationID:   orgID,
		UserID:           userID,
		Name:             name,
		URL:              url,
		SourceType:       "blog",
		CrawlFrequency:   "daily",
		IsEnabled:        false, // Disabled by default, user must approve
		IsDiscovered:     true,
		DiscoveredFromID: &discoveredFromID,
		CrawlStatus:      "pending",
		ContentCount:     0,
		SubscriberCount:  1,
		NextCrawlAt:      &nextCrawlAt,
	}

	if err := s.dataSourceStore.Save(ctx, ds); err != nil {
		return nil, err
	}

	return toResponse(ds), nil
}

// Helper functions

func calculateNextCrawlTime(frequency string) time.Time {
	now := time.Now()
	switch frequency {
	case "hourly":
		return now.Add(time.Hour)
	case "daily":
		return now.Add(24 * time.Hour)
	case "weekly":
		return now.Add(7 * 24 * time.Hour)
	default:
		return now.Add(24 * time.Hour)
	}
}

func toResponse(ds *types.DataSource) *dto.DataSourceResponse {
	return &dto.DataSourceResponse{
		ID:               ds.ID,
		OrganizationID:   ds.OrganizationID,
		UserID:           ds.UserID,
		Name:             ds.Name,
		URL:              ds.URL,
		FeedURL:          ds.FeedURL,
		SourceType:       ds.SourceType,
		CrawlFrequency:   ds.CrawlFrequency,
		IsEnabled:        ds.IsEnabled,
		IsDiscovered:     ds.IsDiscovered,
		DiscoveredFromID: ds.DiscoveredFromID,
		LastCrawledAt:    ds.LastCrawledAt,
		NextCrawlAt:      ds.NextCrawlAt,
		CrawlStatus:      ds.CrawlStatus,
		ErrorMessage:     ds.ErrorMessage,
		ContentCount:     ds.ContentCount,
		SubscriberCount:  ds.SubscriberCount,
		MetaData:         ds.MetaData,
		CreatedAt:        ds.CreatedAt,
		UpdatedAt:        ds.UpdatedAt,
	}
}

func toContentResponse(c *types.CrawledContent) dto.CrawledContentResponse {
	return dto.CrawledContentResponse{
		ID:           c.ID,
		DataSourceID: c.DataSourceID,
		URL:          c.URL,
		Title:        c.Title,
		Content:      c.Content,
		Summary:      c.Summary,
		Author:       c.Author,
		PublishedAt:  c.PublishedAt,
		MetaData:     c.MetaData,
		CreatedAt:    c.CreatedAt,
	}
}
