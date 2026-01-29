package datasource

import (
	"context"
	"time"

	"backend/pkg/core"
	"backend/pkg/database"
	"backend/pkg/database/repository"
	"backend/pkg/types"

	"github.com/google/uuid"
)

// getDataSourceRepo returns a data source repository instance
func getDataSourceRepo() *repository.DataSourceRepository {
	return repository.NewDataSourceRepository(database.DB())
}

// getCrawledContentRepo returns a crawled content repository instance
func getCrawledContentRepo() *repository.CrawledContentRepository {
	return repository.NewCrawledContentRepository(database.DB())
}

// GetByID retrieves a data source by its ID
func GetByID(ctx context.Context, id uuid.UUID) (*types.DataSourceResponse, error) {
	repo := getDataSourceRepo()
	ds, err := repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toResponse(ds), nil
}

// List retrieves all data sources for an organization
func List(ctx context.Context, orgID uuid.UUID) ([]types.DataSourceResponse, error) {
	repo := getDataSourceRepo()
	sources, err := repo.FindByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	result := make([]types.DataSourceResponse, len(sources))
	for i, ds := range sources {
		result[i] = *toResponse(&ds)
	}
	return result, nil
}

// ListByUserID retrieves all data sources for a user (without organization)
func ListByUserID(ctx context.Context, userID uuid.UUID) ([]types.DataSourceResponse, error) {
	repo := getDataSourceRepo()
	sources, err := repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]types.DataSourceResponse, len(sources))
	for i, ds := range sources {
		result[i] = *toResponse(&ds)
	}
	return result, nil
}

// ListAll retrieves all data sources with pagination
func ListAll(ctx context.Context, page, limit int) ([]types.DataSourceResponse, int64, error) {
	repo := getDataSourceRepo()

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	sources, total, err := repo.List(ctx, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	result := make([]types.DataSourceResponse, len(sources))
	for i, ds := range sources {
		result[i] = *toResponse(&ds)
	}
	return result, total, nil
}

// Create creates a new data source
// Either orgID or userID must be provided
func Create(ctx context.Context, orgID *uuid.UUID, userID *uuid.UUID, req types.DataSourceCreateRequest) (*types.DataSourceResponse, error) {
	repo := getDataSourceRepo()

	// Validate that at least one owner is provided
	if orgID == nil && userID == nil {
		return nil, core.InvalidInputError("Either organization_id or user_id must be provided")
	}

	// Check if URL already exists
	existing, err := repo.FindByURL(ctx, req.URL)
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

	if err := repo.Save(ctx, ds); err != nil {
		return nil, err
	}

	return toResponse(ds), nil
}

// Update updates an existing data source
func Update(ctx context.Context, id uuid.UUID, req types.DataSourceUpdateRequest) (*types.DataSourceResponse, error) {
	repo := getDataSourceRepo()

	ds, err := repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if new URL already exists
	if req.URL != nil && *req.URL != ds.URL {
		existing, err := repo.FindByURL(ctx, *req.URL)
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

	if err := repo.Update(ctx, ds); err != nil {
		return nil, err
	}

	return toResponse(ds), nil
}

// Delete removes a data source by its ID
func Delete(ctx context.Context, id uuid.UUID) error {
	repo := getDataSourceRepo()
	return repo.Delete(ctx, id)
}

// TriggerCrawl triggers a manual crawl for a data source
func TriggerCrawl(ctx context.Context, id uuid.UUID) error {
	repo := getDataSourceRepo()

	ds, err := repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Set status to pending and clear next crawl time to prioritize it
	now := time.Now()
	ds.CrawlStatus = "pending"
	ds.NextCrawlAt = &now

	return repo.Update(ctx, ds)
}

// GetContent retrieves crawled content for a data source
func GetContent(ctx context.Context, dataSourceID uuid.UUID, page, limit int) ([]types.CrawledContentResponse, int64, error) {
	contentRepo := getCrawledContentRepo()

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	contents, total, err := contentRepo.FindByDataSourceID(ctx, dataSourceID, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	result := make([]types.CrawledContentResponse, len(contents))
	for i, c := range contents {
		result[i] = toContentResponse(&c)
	}
	return result, total, nil
}

// GetDueToCrawl retrieves data sources that are due for crawling
func GetDueToCrawl(ctx context.Context, limit int) ([]types.DataSource, error) {
	repo := getDataSourceRepo()
	return repo.FindDueToCrawl(ctx, limit)
}

// UpdateCrawlStatus updates the crawl status of a data source
func UpdateCrawlStatus(ctx context.Context, id uuid.UUID, status string, errorMsg *string) error {
	repo := getDataSourceRepo()
	return repo.UpdateCrawlStatus(ctx, id, status, errorMsg)
}

// SetNextCrawlTime sets the next crawl time for a data source
func SetNextCrawlTime(ctx context.Context, id uuid.UUID, frequency string) error {
	repo := getDataSourceRepo()
	nextCrawlAt := calculateNextCrawlTime(frequency)
	return repo.UpdateNextCrawlAt(ctx, id, nextCrawlAt)
}

// CreateDiscoveredSource creates a data source that was discovered automatically
// Either orgID or userID must be provided
func CreateDiscoveredSource(ctx context.Context, orgID *uuid.UUID, userID *uuid.UUID, discoveredFromID uuid.UUID, name, url string) (*types.DataSourceResponse, error) {
	repo := getDataSourceRepo()

	// Validate that at least one owner is provided
	if orgID == nil && userID == nil {
		return nil, core.InvalidInputError("Either organization_id or user_id must be provided")
	}

	// Check if URL already exists
	existing, err := repo.FindByURL(ctx, url)
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

	if err := repo.Save(ctx, ds); err != nil {
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

func toResponse(ds *types.DataSource) *types.DataSourceResponse {
	return &types.DataSourceResponse{
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

func toContentResponse(c *types.CrawledContent) types.CrawledContentResponse {
	return types.CrawledContentResponse{
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
