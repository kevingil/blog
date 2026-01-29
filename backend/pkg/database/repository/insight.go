package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"backend/pkg/core"
	"backend/pkg/database/models"
	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// InsightRepository provides data access for insights
type InsightRepository struct {
	db *gorm.DB
}

// NewInsightRepository creates a new InsightRepository
func NewInsightRepository(db *gorm.DB) *InsightRepository {
	return &InsightRepository{db: db}
}

// insightModelToType converts a database model to types
func insightModelToType(m *models.Insight) *types.Insight {
	var keyPoints []string
	if m.KeyPoints != nil {
		_ = json.Unmarshal(m.KeyPoints, &keyPoints)
	}

	var metaData map[string]interface{}
	if m.MetaData != nil {
		_ = json.Unmarshal(m.MetaData, &metaData)
	}

	var embedding []float32
	if m.Embedding.Slice() != nil {
		embedding = m.Embedding.Slice()
	}

	// Convert string array to UUID array
	sourceContentIDs := make([]uuid.UUID, 0, len(m.SourceContentIDs))
	for _, idStr := range m.SourceContentIDs {
		if id, err := uuid.Parse(idStr); err == nil {
			sourceContentIDs = append(sourceContentIDs, id)
		}
	}

	return &types.Insight{
		ID:               m.ID,
		OrganizationID:   m.OrganizationID,
		TopicID:          m.TopicID,
		Title:            m.Title,
		Summary:          m.Summary,
		Content:          m.Content,
		KeyPoints:        keyPoints,
		SourceContentIDs: sourceContentIDs,
		Embedding:        embedding,
		GeneratedAt:      m.GeneratedAt,
		PeriodStart:      m.PeriodStart,
		PeriodEnd:        m.PeriodEnd,
		IsRead:           m.IsRead,
		IsPinned:         m.IsPinned,
		IsUsedInArticle:  m.IsUsedInArticle,
		MetaData:         metaData,
	}
}

// insightTypeToModel converts a types type to database model
func insightTypeToModel(i *types.Insight) *models.Insight {
	var keyPoints datatypes.JSON
	if i.KeyPoints != nil {
		data, _ := json.Marshal(i.KeyPoints)
		keyPoints = datatypes.JSON(data)
	}

	var metaData datatypes.JSON
	if i.MetaData != nil {
		data, _ := json.Marshal(i.MetaData)
		metaData = datatypes.JSON(data)
	}

	var embedding pgvector.Vector
	if len(i.Embedding) > 0 {
		embedding = pgvector.NewVector(i.Embedding)
	}

	// Convert UUID array to string array
	sourceContentIDs := make(pq.StringArray, 0, len(i.SourceContentIDs))
	for _, id := range i.SourceContentIDs {
		sourceContentIDs = append(sourceContentIDs, id.String())
	}

	return &models.Insight{
		ID:               i.ID,
		OrganizationID:   i.OrganizationID,
		TopicID:          i.TopicID,
		Title:            i.Title,
		Summary:          i.Summary,
		Content:          i.Content,
		KeyPoints:        keyPoints,
		SourceContentIDs: sourceContentIDs,
		Embedding:        embedding,
		GeneratedAt:      i.GeneratedAt,
		PeriodStart:      i.PeriodStart,
		PeriodEnd:        i.PeriodEnd,
		IsRead:           i.IsRead,
		IsPinned:         i.IsPinned,
		IsUsedInArticle:  i.IsUsedInArticle,
		MetaData:         metaData,
	}
}

// FindByID retrieves an insight by its ID
func (r *InsightRepository) FindByID(ctx context.Context, id uuid.UUID) (*types.Insight, error) {
	var model models.Insight
	if err := r.db.WithContext(ctx).Preload("Topic").First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return insightModelToType(&model), nil
}

// List retrieves all insights with pagination
func (r *InsightRepository) List(ctx context.Context, offset, limit int) ([]types.Insight, int64, error) {
	var insightModels []models.Insight
	var total int64

	if err := r.db.WithContext(ctx).Model(&models.Insight{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).
		Preload("Topic").
		Order("generated_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&insightModels).Error; err != nil {
		return nil, 0, err
	}

	insights := make([]types.Insight, len(insightModels))
	for i, m := range insightModels {
		insights[i] = *insightModelToType(&m)
	}
	return insights, total, nil
}

// FindByOrganizationID retrieves all insights for an organization
func (r *InsightRepository) FindByOrganizationID(ctx context.Context, orgID uuid.UUID, offset, limit int) ([]types.Insight, int64, error) {
	var insightModels []models.Insight
	var total int64

	if err := r.db.WithContext(ctx).Model(&models.Insight{}).Where("organization_id = ?", orgID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).
		Preload("Topic").
		Where("organization_id = ?", orgID).
		Order("generated_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&insightModels).Error; err != nil {
		return nil, 0, err
	}

	insights := make([]types.Insight, len(insightModels))
	for i, m := range insightModels {
		insights[i] = *insightModelToType(&m)
	}
	return insights, total, nil
}

// FindByTopicID retrieves all insights for a topic
func (r *InsightRepository) FindByTopicID(ctx context.Context, topicID uuid.UUID, offset, limit int) ([]types.Insight, int64, error) {
	var insightModels []models.Insight
	var total int64

	if err := r.db.WithContext(ctx).Model(&models.Insight{}).Where("topic_id = ?", topicID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).
		Preload("Topic").
		Where("topic_id = ?", topicID).
		Order("generated_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&insightModels).Error; err != nil {
		return nil, 0, err
	}

	insights := make([]types.Insight, len(insightModels))
	for i, m := range insightModels {
		insights[i] = *insightModelToType(&m)
	}
	return insights, total, nil
}

// FindUnread retrieves unread insights for an organization
func (r *InsightRepository) FindUnread(ctx context.Context, orgID uuid.UUID, limit int) ([]types.Insight, error) {
	var insightModels []models.Insight

	if err := r.db.WithContext(ctx).
		Preload("Topic").
		Where("organization_id = ? AND is_read = ?", orgID, false).
		Order("generated_at DESC").
		Limit(limit).
		Find(&insightModels).Error; err != nil {
		return nil, err
	}

	insights := make([]types.Insight, len(insightModels))
	for i, m := range insightModels {
		insights[i] = *insightModelToType(&m)
	}
	return insights, nil
}

// SearchSimilar performs vector similarity search for insights
func (r *InsightRepository) SearchSimilar(ctx context.Context, embedding []float32, limit int) ([]types.Insight, error) {
	var insightModels []models.Insight

	embeddingVector := pgvector.NewVector(embedding)
	query := fmt.Sprintf(
		"SELECT * FROM insight WHERE embedding IS NOT NULL ORDER BY embedding <=> '%s' LIMIT %d",
		embeddingVector.String(),
		limit,
	)

	if err := r.db.WithContext(ctx).Raw(query).Scan(&insightModels).Error; err != nil {
		return nil, err
	}

	insights := make([]types.Insight, len(insightModels))
	for i, m := range insightModels {
		insights[i] = *insightModelToType(&m)
	}
	return insights, nil
}

// SearchSimilarByOrg performs vector similarity search for insights within an organization
func (r *InsightRepository) SearchSimilarByOrg(ctx context.Context, orgID uuid.UUID, embedding []float32, limit int) ([]types.Insight, error) {
	var insightModels []models.Insight

	embeddingVector := pgvector.NewVector(embedding)
	query := fmt.Sprintf(
		"SELECT * FROM insight WHERE organization_id = '%s' AND embedding IS NOT NULL ORDER BY embedding <=> '%s' LIMIT %d",
		orgID.String(),
		embeddingVector.String(),
		limit,
	)

	if err := r.db.WithContext(ctx).Raw(query).Scan(&insightModels).Error; err != nil {
		return nil, err
	}

	insights := make([]types.Insight, len(insightModels))
	for i, m := range insightModels {
		insights[i] = *insightModelToType(&m)
	}
	return insights, nil
}

// Save creates a new insight
func (r *InsightRepository) Save(ctx context.Context, insight *types.Insight) error {
	model := insightTypeToModel(insight)
	if model.ID == uuid.Nil {
		model.ID = uuid.New()
		insight.ID = model.ID
	}
	return r.db.WithContext(ctx).Create(model).Error
}

// Update updates an existing insight
func (r *InsightRepository) Update(ctx context.Context, insight *types.Insight) error {
	model := insightTypeToModel(insight)
	return r.db.WithContext(ctx).Save(model).Error
}

// MarkAsRead marks an insight as read
func (r *InsightRepository) MarkAsRead(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&models.Insight{}).
		Where("id = ?", id).
		Update("is_read", true).Error
}

// TogglePinned toggles the pinned status of an insight
func (r *InsightRepository) TogglePinned(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&models.Insight{}).
		Where("id = ?", id).
		Update("is_pinned", gorm.Expr("NOT is_pinned")).Error
}

// MarkAsUsedInArticle marks an insight as used in an article
func (r *InsightRepository) MarkAsUsedInArticle(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&models.Insight{}).
		Where("id = ?", id).
		Update("is_used_in_article", true).Error
}

// Delete removes an insight by its ID
func (r *InsightRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Insight{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.ErrNotFound
	}
	return nil
}

// CountUnread counts unread insights for an organization
func (r *InsightRepository) CountUnread(ctx context.Context, orgID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Insight{}).
		Where("organization_id = ? AND is_read = ?", orgID, false).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountAllUnread returns the total count of unread insights (no org filter)
func (r *InsightRepository) CountAllUnread(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Insight{}).
		Where("is_read = ?", false).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
