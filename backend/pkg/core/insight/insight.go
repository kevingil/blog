// Package insight provides the Insight domain types and store interfaces
package insight

import (
	"context"
	"time"

	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// Insight is an alias to types.Insight for backward compatibility
type Insight = types.Insight

// InsightTopic is an alias to types.InsightTopic for backward compatibility
type InsightTopic = types.InsightTopic

// InsightStore defines the data access interface for insights
type InsightStore interface {
	FindByID(ctx context.Context, id uuid.UUID) (*types.Insight, error)
	List(ctx context.Context, offset, limit int) ([]types.Insight, int64, error)
	FindByOrganizationID(ctx context.Context, orgID uuid.UUID, offset, limit int) ([]types.Insight, int64, error)
	FindByTopicID(ctx context.Context, topicID uuid.UUID, offset, limit int) ([]types.Insight, int64, error)
	FindUnread(ctx context.Context, orgID uuid.UUID, limit int) ([]types.Insight, error)
	SearchSimilar(ctx context.Context, embedding []float32, limit int) ([]types.Insight, error)
	SearchSimilarByOrg(ctx context.Context, orgID uuid.UUID, embedding []float32, limit int) ([]types.Insight, error)
	Save(ctx context.Context, insight *types.Insight) error
	MarkAsRead(ctx context.Context, id uuid.UUID) error
	TogglePinned(ctx context.Context, id uuid.UUID) error
	MarkAsUsedInArticle(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	CountUnread(ctx context.Context, orgID uuid.UUID) (int64, error)
	CountAllUnread(ctx context.Context) (int64, error)
}

// InsightTopicStore defines the data access interface for insight topics
type InsightTopicStore interface {
	FindByID(ctx context.Context, id uuid.UUID) (*types.InsightTopic, error)
	FindByOrganizationID(ctx context.Context, orgID uuid.UUID) ([]types.InsightTopic, error)
	FindAll(ctx context.Context) ([]types.InsightTopic, error)
	SearchSimilar(ctx context.Context, embedding []float32, limit int, threshold float64) ([]types.InsightTopic, []float64, error)
	Save(ctx context.Context, topic *types.InsightTopic) error
	Update(ctx context.Context, topic *types.InsightTopic) error
	UpdateContentCount(ctx context.Context, id uuid.UUID, count int) error
	UpdateLastInsightAt(ctx context.Context, id uuid.UUID, timestamp time.Time) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// UserInsightStatusStore defines the data access interface for user insight status
type UserInsightStatusStore interface {
	FindByUserAndInsight(ctx context.Context, userID, insightID uuid.UUID) (*types.UserInsightStatus, error)
	MarkAsRead(ctx context.Context, userID, insightID uuid.UUID) error
	TogglePinned(ctx context.Context, userID, insightID uuid.UUID) (bool, error)
	MarkAsUsedInArticle(ctx context.Context, userID, insightID uuid.UUID) error
	GetStatusMapForInsights(ctx context.Context, userID uuid.UUID, insightIDs []uuid.UUID) (map[uuid.UUID]*types.UserInsightStatus, error)
	CountUnreadByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
}

// InsightCrawledContentStore defines the data access interface for crawled content (insight-specific operations)
type InsightCrawledContentStore interface {
	FindByIDs(ctx context.Context, ids []uuid.UUID) ([]types.CrawledContent, error)
	SearchSimilar(ctx context.Context, embedding []float32, limit int) ([]types.CrawledContent, error)
	SearchSimilarByOrg(ctx context.Context, orgID uuid.UUID, embedding []float32, limit int) ([]types.CrawledContent, error)
	FindRecentByOrg(ctx context.Context, orgID uuid.UUID, limit int) ([]types.CrawledContent, error)
}

// ContentTopicMatchStore defines the data access interface for content-topic matches
type ContentTopicMatchStore interface {
	SaveBatch(ctx context.Context, matches []types.ContentTopicMatch) error
	CountByTopicID(ctx context.Context, topicID uuid.UUID) (int64, error)
}

// EmbeddingGenerator defines the interface for generating embeddings
type EmbeddingGenerator interface {
	GenerateEmbedding(ctx context.Context, text string) (pgvector.Vector, error)
}
