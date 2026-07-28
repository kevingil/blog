package repository

import (
	"context"

	"backend/pkg/database/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ChatMessageRepository implements data access for chat messages using GORM
type ChatMessageRepository struct {
	db *gorm.DB
}

// NewChatMessageRepository creates a new ChatMessageRepository
func NewChatMessageRepository(db *gorm.DB) *ChatMessageRepository {
	return &ChatMessageRepository{db: db}
}

// Create inserts a new chat message into the database
func (r *ChatMessageRepository) Create(ctx context.Context, message *models.ChatMessage) error {
	return r.db.WithContext(ctx).Create(message).Error
}

// GetByID retrieves a chat message by its ID
func (r *ChatMessageRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ChatMessage, error) {
	var message models.ChatMessage
	err := r.db.WithContext(ctx).First(&message, id).Error
	if err != nil {
		return nil, err
	}
	return &message, nil
}

// GetByArticleID retrieves chat messages for an article with limit, ordered by created_at DESC
func (r *ChatMessageRepository) GetByArticleID(ctx context.Context, articleID uuid.UUID, limit int) ([]models.ChatMessage, error) {
	var messages []models.ChatMessage
	err := r.db.WithContext(ctx).
		Where("article_id = ?", articleID).
		Order("created_at DESC").
		Limit(limit).
		Find(&messages).Error
	if err != nil {
		return nil, err
	}
	return messages, nil
}

// GetPendingArtifacts retrieves messages with pending artifacts for an article
func (r *ChatMessageRepository) GetPendingArtifacts(ctx context.Context, articleID uuid.UUID) ([]models.ChatMessage, error) {
	var messages []models.ChatMessage
	err := r.db.WithContext(ctx).
		Where("article_id = ? AND meta_data->'artifact'->>'status' = ?", articleID, "pending").
		Order("created_at DESC").
		Find(&messages).Error
	if err != nil {
		return nil, err
	}
	return messages, nil
}

// Update saves changes to an existing chat message
func (r *ChatMessageRepository) Update(ctx context.Context, message *models.ChatMessage) error {
	return r.db.WithContext(ctx).Save(message).Error
}

// UpdateMetaData updates only the meta_data field of a chat message
func (r *ChatMessageRepository) UpdateMetaData(ctx context.Context, id uuid.UUID, metaData []byte) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&models.ChatMessage{}).
		Where("id = ?", id).
		Update("meta_data", metaData)
	return result.RowsAffected, result.Error
}

// DeleteByArticleID removes all chat messages for an article
func (r *ChatMessageRepository) DeleteByArticleID(ctx context.Context, articleID uuid.UUID) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("article_id = ?", articleID).
		Delete(&models.ChatMessage{})
	return result.RowsAffected, result.Error
}
