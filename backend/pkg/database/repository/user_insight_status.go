package repository

import (
	"context"
	"time"

	"backend/pkg/database/models"
	"backend/pkg/types"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserInsightStatusRepository provides data access for user insight status
type UserInsightStatusRepository struct {
	db *gorm.DB
}

// NewUserInsightStatusRepository creates a new UserInsightStatusRepository
func NewUserInsightStatusRepository(db *gorm.DB) *UserInsightStatusRepository {
	return &UserInsightStatusRepository{db: db}
}

// userInsightStatusModelToType converts a database model to types
func userInsightStatusModelToType(m *models.UserInsightStatus) *types.UserInsightStatus {
	return &types.UserInsightStatus{
		ID:              m.ID,
		UserID:          m.UserID,
		InsightID:       m.InsightID,
		IsRead:          m.IsRead,
		IsPinned:        m.IsPinned,
		IsUsedInArticle: m.IsUsedInArticle,
		ReadAt:          m.ReadAt,
		CreatedAt:       m.CreatedAt,
	}
}

// userInsightStatusTypeToModel converts a types type to database model
func userInsightStatusTypeToModel(s *types.UserInsightStatus) *models.UserInsightStatus {
	return &models.UserInsightStatus{
		ID:              s.ID,
		UserID:          s.UserID,
		InsightID:       s.InsightID,
		IsRead:          s.IsRead,
		IsPinned:        s.IsPinned,
		IsUsedInArticle: s.IsUsedInArticle,
		ReadAt:          s.ReadAt,
		CreatedAt:       s.CreatedAt,
	}
}

// FindByUserAndInsight retrieves a user insight status by user and insight IDs
func (r *UserInsightStatusRepository) FindByUserAndInsight(ctx context.Context, userID, insightID uuid.UUID) (*types.UserInsightStatus, error) {
	var model models.UserInsightStatus
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND insight_id = ?", userID, insightID).
		First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // Return nil, nil for not found (different from error)
		}
		return nil, err
	}
	return userInsightStatusModelToType(&model), nil
}

// FindByUserID retrieves all insight statuses for a user
func (r *UserInsightStatusRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]types.UserInsightStatus, error) {
	var statusModels []models.UserInsightStatus
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&statusModels).Error; err != nil {
		return nil, err
	}

	statuses := make([]types.UserInsightStatus, len(statusModels))
	for i, m := range statusModels {
		statuses[i] = *userInsightStatusModelToType(&m)
	}
	return statuses, nil
}

// FindUnreadByUserID retrieves unread insight statuses for a user
func (r *UserInsightStatusRepository) FindUnreadByUserID(ctx context.Context, userID uuid.UUID) ([]types.UserInsightStatus, error) {
	var statusModels []models.UserInsightStatus
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_read = ?", userID, false).
		Find(&statusModels).Error; err != nil {
		return nil, err
	}

	statuses := make([]types.UserInsightStatus, len(statusModels))
	for i, m := range statusModels {
		statuses[i] = *userInsightStatusModelToType(&m)
	}
	return statuses, nil
}

// FindPinnedByUserID retrieves pinned insight statuses for a user
func (r *UserInsightStatusRepository) FindPinnedByUserID(ctx context.Context, userID uuid.UUID) ([]types.UserInsightStatus, error) {
	var statusModels []models.UserInsightStatus
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_pinned = ?", userID, true).
		Find(&statusModels).Error; err != nil {
		return nil, err
	}

	statuses := make([]types.UserInsightStatus, len(statusModels))
	for i, m := range statusModels {
		statuses[i] = *userInsightStatusModelToType(&m)
	}
	return statuses, nil
}

// Upsert creates or updates a user insight status
func (r *UserInsightStatusRepository) Upsert(ctx context.Context, status *types.UserInsightStatus) error {
	model := userInsightStatusTypeToModel(status)
	if model.ID == uuid.Nil {
		model.ID = uuid.New()
		status.ID = model.ID
	}

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "insight_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"is_read", "is_pinned", "is_used_in_article", "read_at"}),
	}).Create(model).Error
}

// MarkAsRead marks an insight as read for a user
func (r *UserInsightStatusRepository) MarkAsRead(ctx context.Context, userID, insightID uuid.UUID) error {
	now := time.Now()
	status := &types.UserInsightStatus{
		UserID:    userID,
		InsightID: insightID,
		IsRead:    true,
		ReadAt:    &now,
	}
	return r.Upsert(ctx, status)
}

// TogglePinned toggles the pinned status for a user's insight
func (r *UserInsightStatusRepository) TogglePinned(ctx context.Context, userID, insightID uuid.UUID) (bool, error) {
	// First, try to find existing status
	existing, err := r.FindByUserAndInsight(ctx, userID, insightID)
	if err != nil {
		return false, err
	}

	newPinned := true
	if existing != nil {
		newPinned = !existing.IsPinned
	}

	status := &types.UserInsightStatus{
		UserID:    userID,
		InsightID: insightID,
		IsPinned:  newPinned,
	}
	if existing != nil {
		status.ID = existing.ID
		status.IsRead = existing.IsRead
		status.IsUsedInArticle = existing.IsUsedInArticle
		status.ReadAt = existing.ReadAt
	}

	return newPinned, r.Upsert(ctx, status)
}

// MarkAsUsedInArticle marks an insight as used in an article for a user
func (r *UserInsightStatusRepository) MarkAsUsedInArticle(ctx context.Context, userID, insightID uuid.UUID) error {
	existing, err := r.FindByUserAndInsight(ctx, userID, insightID)
	if err != nil {
		return err
	}

	status := &types.UserInsightStatus{
		UserID:          userID,
		InsightID:       insightID,
		IsUsedInArticle: true,
	}
	if existing != nil {
		status.ID = existing.ID
		status.IsRead = existing.IsRead
		status.IsPinned = existing.IsPinned
		status.ReadAt = existing.ReadAt
	}

	return r.Upsert(ctx, status)
}

// CountUnreadByUserID counts unread insights for a user
func (r *UserInsightStatusRepository) CountUnreadByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	// Count insights that don't have a read status or have is_read = false
	// This requires a subquery approach since we want insights without any status too
	if err := r.db.WithContext(ctx).
		Model(&models.UserInsightStatus{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// GetStatusMapForInsights returns a map of insight ID to user status for multiple insights
func (r *UserInsightStatusRepository) GetStatusMapForInsights(ctx context.Context, userID uuid.UUID, insightIDs []uuid.UUID) (map[uuid.UUID]*types.UserInsightStatus, error) {
	if len(insightIDs) == 0 {
		return make(map[uuid.UUID]*types.UserInsightStatus), nil
	}

	var statusModels []models.UserInsightStatus
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND insight_id IN ?", userID, insightIDs).
		Find(&statusModels).Error; err != nil {
		return nil, err
	}

	statusMap := make(map[uuid.UUID]*types.UserInsightStatus)
	for _, m := range statusModels {
		status := userInsightStatusModelToType(&m)
		statusMap[m.InsightID] = status
	}
	return statusMap, nil
}

// Delete removes a user insight status
func (r *UserInsightStatusRepository) Delete(ctx context.Context, userID, insightID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND insight_id = ?", userID, insightID).
		Delete(&models.UserInsightStatus{}).Error
}
