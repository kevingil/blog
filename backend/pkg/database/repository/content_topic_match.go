package repository

import (
	"context"

	"backend/pkg/core"
	"backend/pkg/database/models"
	"backend/pkg/types"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ContentTopicMatchRepository provides data access for content-topic matches
type ContentTopicMatchRepository struct {
	db *gorm.DB
}

// NewContentTopicMatchRepository creates a new ContentTopicMatchRepository
func NewContentTopicMatchRepository(db *gorm.DB) *ContentTopicMatchRepository {
	return &ContentTopicMatchRepository{db: db}
}

// contentTopicMatchModelToType converts a database model to types
func contentTopicMatchModelToType(m *models.ContentTopicMatch) *types.ContentTopicMatch {
	return &types.ContentTopicMatch{
		ID:              m.ID,
		ContentID:       m.ContentID,
		TopicID:         m.TopicID,
		SimilarityScore: m.SimilarityScore,
		IsPrimary:       m.IsPrimary,
		CreatedAt:       m.CreatedAt,
	}
}

// contentTopicMatchTypeToModel converts a types type to database model
func contentTopicMatchTypeToModel(m *types.ContentTopicMatch) *models.ContentTopicMatch {
	return &models.ContentTopicMatch{
		ID:              m.ID,
		ContentID:       m.ContentID,
		TopicID:         m.TopicID,
		SimilarityScore: m.SimilarityScore,
		IsPrimary:       m.IsPrimary,
		CreatedAt:       m.CreatedAt,
	}
}

// FindByContentID retrieves all topic matches for a content item
func (r *ContentTopicMatchRepository) FindByContentID(ctx context.Context, contentID uuid.UUID) ([]types.ContentTopicMatch, error) {
	var matchModels []models.ContentTopicMatch
	if err := r.db.WithContext(ctx).
		Where("content_id = ?", contentID).
		Order("similarity_score DESC").
		Find(&matchModels).Error; err != nil {
		return nil, err
	}

	matches := make([]types.ContentTopicMatch, len(matchModels))
	for i, m := range matchModels {
		matches[i] = *contentTopicMatchModelToType(&m)
	}
	return matches, nil
}

// FindByTopicID retrieves all content matches for a topic
func (r *ContentTopicMatchRepository) FindByTopicID(ctx context.Context, topicID uuid.UUID, offset, limit int) ([]types.ContentTopicMatch, int64, error) {
	var matchModels []models.ContentTopicMatch
	var total int64

	if err := r.db.WithContext(ctx).Model(&models.ContentTopicMatch{}).Where("topic_id = ?", topicID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).
		Where("topic_id = ?", topicID).
		Order("similarity_score DESC").
		Offset(offset).
		Limit(limit).
		Find(&matchModels).Error; err != nil {
		return nil, 0, err
	}

	matches := make([]types.ContentTopicMatch, len(matchModels))
	for i, m := range matchModels {
		matches[i] = *contentTopicMatchModelToType(&m)
	}
	return matches, total, nil
}

// FindPrimaryByTopicID retrieves all primary matches for a topic
func (r *ContentTopicMatchRepository) FindPrimaryByTopicID(ctx context.Context, topicID uuid.UUID, offset, limit int) ([]types.ContentTopicMatch, int64, error) {
	var matchModels []models.ContentTopicMatch
	var total int64

	if err := r.db.WithContext(ctx).Model(&models.ContentTopicMatch{}).
		Where("topic_id = ? AND is_primary = ?", topicID, true).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).
		Where("topic_id = ? AND is_primary = ?", topicID, true).
		Order("similarity_score DESC").
		Offset(offset).
		Limit(limit).
		Find(&matchModels).Error; err != nil {
		return nil, 0, err
	}

	matches := make([]types.ContentTopicMatch, len(matchModels))
	for i, m := range matchModels {
		matches[i] = *contentTopicMatchModelToType(&m)
	}
	return matches, total, nil
}

// Save creates or updates a content-topic match (upserts)
func (r *ContentTopicMatchRepository) Save(ctx context.Context, match *types.ContentTopicMatch) error {
	model := contentTopicMatchTypeToModel(match)
	if model.ID == uuid.Nil {
		model.ID = uuid.New()
		match.ID = model.ID
	}

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "content_id"}, {Name: "topic_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"similarity_score", "is_primary"}),
	}).Create(model).Error
}

// SaveBatch creates or updates multiple content-topic matches
func (r *ContentTopicMatchRepository) SaveBatch(ctx context.Context, matches []types.ContentTopicMatch) error {
	if len(matches) == 0 {
		return nil
	}

	matchModels := make([]models.ContentTopicMatch, len(matches))
	for i, m := range matches {
		if m.ID == uuid.Nil {
			matches[i].ID = uuid.New()
		}
		matchModels[i] = *contentTopicMatchTypeToModel(&matches[i])
	}

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "content_id"}, {Name: "topic_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"similarity_score", "is_primary"}),
	}).Create(&matchModels).Error
}

// DeleteByContentID removes all topic matches for a content item
func (r *ContentTopicMatchRepository) DeleteByContentID(ctx context.Context, contentID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("content_id = ?", contentID).Delete(&models.ContentTopicMatch{}).Error
}

// DeleteByTopicID removes all content matches for a topic
func (r *ContentTopicMatchRepository) DeleteByTopicID(ctx context.Context, topicID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("topic_id = ?", topicID).Delete(&models.ContentTopicMatch{}).Error
}

// Delete removes a specific content-topic match
func (r *ContentTopicMatchRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.ContentTopicMatch{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.ErrNotFound
	}
	return nil
}

// CountByTopicID counts matches for a topic
func (r *ContentTopicMatchRepository) CountByTopicID(ctx context.Context, topicID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.ContentTopicMatch{}).
		Where("topic_id = ?", topicID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// UpdatePrimaryStatus sets the is_primary flag for a match
func (r *ContentTopicMatchRepository) UpdatePrimaryStatus(ctx context.Context, contentID uuid.UUID, topicID uuid.UUID, isPrimary bool) error {
	return r.db.WithContext(ctx).Model(&models.ContentTopicMatch{}).
		Where("content_id = ? AND topic_id = ?", contentID, topicID).
		Update("is_primary", isPrimary).Error
}

// ClearPrimaryForContent clears primary flag for all matches of a content item
func (r *ContentTopicMatchRepository) ClearPrimaryForContent(ctx context.Context, contentID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&models.ContentTopicMatch{}).
		Where("content_id = ?", contentID).
		Update("is_primary", false).Error
}
