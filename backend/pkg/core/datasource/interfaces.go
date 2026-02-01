package datasource

import (
	"context"
	"time"

	"backend/pkg/types"

	"github.com/google/uuid"
)

// DataSourceStore defines the interface for data source persistence operations
type DataSourceStore interface {
	FindByID(ctx context.Context, id uuid.UUID) (*types.DataSource, error)
	FindByOrganizationID(ctx context.Context, orgID uuid.UUID) ([]types.DataSource, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]types.DataSource, error)
	FindByURL(ctx context.Context, url string) (*types.DataSource, error)
	FindDueToCrawl(ctx context.Context, limit int) ([]types.DataSource, error)
	List(ctx context.Context, offset, limit int) ([]types.DataSource, int64, error)
	Save(ctx context.Context, source *types.DataSource) error
	Update(ctx context.Context, source *types.DataSource) error
	UpdateCrawlStatus(ctx context.Context, id uuid.UUID, status string, errorMsg *string) error
	UpdateNextCrawlAt(ctx context.Context, id uuid.UUID, nextCrawlAt time.Time) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// CrawledContentStore defines the interface for crawled content persistence operations
type CrawledContentStore interface {
	FindByDataSourceID(ctx context.Context, dsID uuid.UUID, offset, limit int) ([]types.CrawledContent, int64, error)
}
