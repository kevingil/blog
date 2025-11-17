// Package chat provides chat message persistence and management
package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"blog-agent-go/backend/internal/core/agent/metadata"
	"blog-agent-go/backend/internal/database"
	"blog-agent-go/backend/internal/errors"
	"blog-agent-go/backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// MessageService provides operations for managing chat messages
type MessageService struct {
	db database.Service
}

// NewMessageService creates a new chat message service
func NewMessageService(db database.Service) *MessageService {
	return &MessageService{
		db: db,
	}
}

// SaveMessageRequest represents a request to save a chat message
type SaveMessageRequest struct {
	ArticleID uuid.UUID
	Role      string
	Content   string
	MetaData  *metadata.MessageMetaData
}

// SaveMessage saves a new chat message to the database
func (s *MessageService) SaveMessage(ctx context.Context, req SaveMessageRequest) (*models.ChatMessage, error) {
	// Marshal metadata to JSON
	var metaDataJSON datatypes.JSON
	if req.MetaData != nil {
		// Validate metadata
		if err := metadata.ValidateMetaData(req.MetaData); err != nil {
			return nil, errors.NewValidationError(fmt.Sprintf("Invalid metadata: %v", err))
		}
		
		jsonBytes, err := json.Marshal(req.MetaData)
		if err != nil {
			return nil, errors.NewInternalError("Failed to marshal metadata")
		}
		metaDataJSON = datatypes.JSON(jsonBytes)
	} else {
		metaDataJSON = datatypes.JSON("{}")
	}

	message := &models.ChatMessage{
		ArticleID: req.ArticleID,
		Role:      req.Role,
		Content:   req.Content,
		MetaData:  metaDataJSON,
	}

	db := s.db.GetDB()
	if err := db.Create(message).Error; err != nil {
		return nil, errors.NewInternalError("Failed to save message")
	}

	return message, nil
}

// GetConversationHistory retrieves chat messages for an article
func (s *MessageService) GetConversationHistory(ctx context.Context, articleID uuid.UUID, limit int) ([]models.ChatMessage, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200 // Cap at 200 messages
	}

	db := s.db.GetDB()
	var messages []models.ChatMessage

	err := db.Where("article_id = ?", articleID).
		Order("created_at ASC").
		Limit(limit).
		Find(&messages).Error

	if err != nil {
		return nil, errors.NewInternalError("Failed to retrieve conversation history")
	}

	return messages, nil
}

// UpdateMessageMetadata updates the metadata of an existing message
func (s *MessageService) UpdateMessageMetadata(ctx context.Context, messageID uuid.UUID, metaData *metadata.MessageMetaData) error {
	// Validate metadata
	if err := metadata.ValidateMetaData(metaData); err != nil {
		return errors.NewValidationError(fmt.Sprintf("Invalid metadata: %v", err))
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(metaData)
	if err != nil {
		return errors.NewInternalError("Failed to marshal metadata")
	}

	db := s.db.GetDB()
	result := db.Model(&models.ChatMessage{}).
		Where("id = ?", messageID).
		Update("meta_data", datatypes.JSON(jsonBytes))

	if result.Error != nil {
		return errors.NewInternalError("Failed to update message metadata")
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("Message")
	}

	return nil
}

// UpdateArtifactStatus updates the status of an artifact within a message
func (s *MessageService) UpdateArtifactStatus(ctx context.Context, messageID uuid.UUID, artifactID, status string) error {
	db := s.db.GetDB()

	// Get the current message
	var message models.ChatMessage
	if err := db.First(&message, messageID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewNotFoundError("Message")
		}
		return errors.NewInternalError("Failed to retrieve message")
	}

	// Parse current metadata
	var metaData metadata.MessageMetaData
	if len(message.MetaData) > 0 && string(message.MetaData) != "{}" {
		if err := json.Unmarshal(message.MetaData, &metaData); err != nil {
			return errors.NewInternalError("Failed to parse message metadata")
		}
	}

	// Update artifact status
	if metaData.Artifact == nil {
		return errors.NewNotFoundError("Artifact")
	}

	if artifactID != "" && metaData.Artifact.ID != artifactID {
		return errors.NewNotFoundError("Artifact")
	}

	metaData.Artifact.Status = status
	if status == metadata.ArtifactStatusAccepted || status == metadata.ArtifactStatusApplied {
		now := time.Now()
		metaData.Artifact.AppliedAt = &now
	}

	// Save updated metadata
	return s.UpdateMessageMetadata(ctx, messageID, &metaData)
}

// GetMessageByID retrieves a single message by ID
func (s *MessageService) GetMessageByID(ctx context.Context, messageID uuid.UUID) (*models.ChatMessage, error) {
	db := s.db.GetDB()
	var message models.ChatMessage

	if err := db.First(&message, messageID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("Message")
		}
		return nil, errors.NewInternalError("Failed to retrieve message")
	}

	return &message, nil
}

// GetPendingArtifacts retrieves all messages with pending artifacts for an article
func (s *MessageService) GetPendingArtifacts(ctx context.Context, articleID uuid.UUID) ([]models.ChatMessage, error) {
	db := s.db.GetDB()
	var messages []models.ChatMessage

	// Query for messages where meta_data->'artifact'->>'status' = 'pending'
	err := db.Where("article_id = ? AND meta_data->'artifact'->>'status' = ?", articleID, metadata.ArtifactStatusPending).
		Order("created_at DESC").
		Find(&messages).Error

	if err != nil {
		return nil, errors.NewInternalError("Failed to retrieve pending artifacts")
	}

	return messages, nil
}

// AcceptArtifact marks an artifact as accepted
func (s *MessageService) AcceptArtifact(ctx context.Context, messageID uuid.UUID, feedback string) error {
	// Update artifact status
	if err := s.UpdateArtifactStatus(ctx, messageID, "", metadata.ArtifactStatusAccepted); err != nil {
		return err
	}

	// Add user action to metadata
	message, err := s.GetMessageByID(ctx, messageID)
	if err != nil {
		return err
	}

	var metaData metadata.MessageMetaData
	if len(message.MetaData) > 0 && string(message.MetaData) != "{}" {
		if err := json.Unmarshal(message.MetaData, &metaData); err != nil {
			return errors.NewInternalError("Failed to parse metadata")
		}
	}

	// Add user action
	userAction := metadata.NewUserAction(metadata.UserActionAccept, "", feedback, "")
	if metaData.Artifact != nil {
		userAction.ArtifactID = metaData.Artifact.ID
	}
	metaData.UserAction = userAction

	return s.UpdateMessageMetadata(ctx, messageID, &metaData)
}

// RejectArtifact marks an artifact as rejected
func (s *MessageService) RejectArtifact(ctx context.Context, messageID uuid.UUID, reason string) error {
	// Update artifact status
	if err := s.UpdateArtifactStatus(ctx, messageID, "", metadata.ArtifactStatusRejected); err != nil {
		return err
	}

	// Add user action to metadata
	message, err := s.GetMessageByID(ctx, messageID)
	if err != nil {
		return err
	}

	var metaData metadata.MessageMetaData
	if len(message.MetaData) > 0 && string(message.MetaData) != "{}" {
		if err := json.Unmarshal(message.MetaData, &metaData); err != nil {
			return errors.NewInternalError("Failed to parse metadata")
		}
	}

	// Add user action
	userAction := metadata.NewUserAction(metadata.UserActionReject, "", "", reason)
	if metaData.Artifact != nil {
		userAction.ArtifactID = metaData.Artifact.ID
	}
	metaData.UserAction = userAction

	return s.UpdateMessageMetadata(ctx, messageID, &metaData)
}

