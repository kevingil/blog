// Package chat provides chat message persistence and management
package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

// SaveMessage saves a new chat message to the database
func (s *MessageService) SaveMessage(ctx context.Context, articleID uuid.UUID, role, content string, metaData *metadata.MessageMetaData) (*models.ChatMessage, error) {
	log.Printf("[MessageService] 💾 SaveMessage called")
	log.Printf("[MessageService]    Article ID: %s", articleID)
	log.Printf("[MessageService]    Role: %s", role)
	log.Printf("[MessageService]    Content length: %d chars", len(content))

	// Marshal metadata to JSON
	var metaDataJSON datatypes.JSON
	if metaData != nil {
		// Validate metadata
		if err := metadata.ValidateMetaData(metaData); err != nil {
			log.Printf("[MessageService] ❌ Metadata validation failed: %v", err)
			return nil, errors.NewValidationError(fmt.Sprintf("Invalid metadata: %v", err))
		}

		jsonBytes, err := json.Marshal(metaData)
		if err != nil {
			log.Printf("[MessageService] ❌ Failed to marshal metadata: %v", err)
			return nil, errors.NewInternalError("Failed to marshal metadata")
		}
		metaDataJSON = datatypes.JSON(jsonBytes)
		log.Printf("[MessageService]    Metadata: %d bytes", len(jsonBytes))

		// Log artifact if present
		if metaData.Artifact != nil {
			log.Printf("[MessageService]    📋 Contains ARTIFACT:")
			log.Printf("[MessageService]       ID: %s", metaData.Artifact.ID)
			log.Printf("[MessageService]       Type: %s", metaData.Artifact.Type)
			log.Printf("[MessageService]       Status: %s", metaData.Artifact.Status)
		}

		// Log tool execution if present
		if metaData.ToolExecution != nil {
			log.Printf("[MessageService]    🔧 Contains TOOL EXECUTION:")
			log.Printf("[MessageService]       Tool: %s", metaData.ToolExecution.ToolName)
			log.Printf("[MessageService]       Success: %v", metaData.ToolExecution.Success)
		}
	} else {
		metaDataJSON = datatypes.JSON("{}")
		log.Printf("[MessageService]    Metadata: none")
	}

	message := &models.ChatMessage{
		ArticleID: articleID,
		Role:      role,
		Content:   content,
		MetaData:  metaDataJSON,
	}

	db := s.db.GetDB()
	if err := db.Create(message).Error; err != nil {
		log.Printf("[MessageService] ❌ Database INSERT failed: %v", err)
		return nil, errors.NewInternalError("Failed to save message")
	}

	log.Printf("[MessageService] ✅ Message saved successfully (ID: %s)", message.ID)

	return message, nil
}

// Default initial message for new conversations
const initialGreeting = "Hi! I can help you improve your article. Try asking me to \"rewrite the introduction\" or \"make the content more engaging\"."

// GetConversationHistory retrieves chat messages for an article
func (s *MessageService) GetConversationHistory(ctx context.Context, articleID uuid.UUID, limit int) ([]models.ChatMessage, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200 // Cap at 200 messages
	}

	fmt.Printf("[MessageService] 🔍 Querying database for messages...\n")
	fmt.Printf("[MessageService]    Article ID: %s\n", articleID)
	fmt.Printf("[MessageService]    Limit: %d\n", limit)

	db := s.db.GetDB()
	var messages []models.ChatMessage

	err := db.Where("article_id = ?", articleID).
		Order("created_at ASC").
		Limit(limit).
		Find(&messages).Error

	if err != nil {
		fmt.Printf("[MessageService] ❌ Database query failed: %v\n", err)
		return nil, errors.NewInternalError("Failed to retrieve conversation history")
	}

	fmt.Printf("[MessageService] ✅ Database returned %d messages\n", len(messages))

	// If no messages exist, return a default initial greeting
	if len(messages) == 0 {
		fmt.Printf("[MessageService] 📝 No messages found, returning initial greeting\n")

		initialMessage := models.ChatMessage{
			ID:        uuid.New(),
			ArticleID: articleID,
			Role:      "assistant",
			Content:   initialGreeting,
			MetaData:  datatypes.JSON("{}"),
			CreatedAt: time.Now(),
		}

		return []models.ChatMessage{initialMessage}, nil
	}

	// Log each message from database
	for i, msg := range messages {
		preview := msg.Content
		if len(preview) > 50 {
			preview = preview[:50] + "..."
		}
		fmt.Printf("[MessageService]    [%d] %s (ID: %s): %s\n", i+1, msg.Role, msg.ID, preview)

		// Check if metadata is present
		metadataSize := len(msg.MetaData)
		if metadataSize > 2 {
			fmt.Printf("[MessageService]        Metadata: %d bytes\n", metadataSize)
		}
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

// GetArtifactContent retrieves the artifact content from a message
func (s *MessageService) GetArtifactContent(ctx context.Context, messageID uuid.UUID) (string, error) {
	message, err := s.GetMessageByID(ctx, messageID)
	if err != nil {
		return "", err
	}

	// Parse metadata
	var metaData metadata.MessageMetaData
	if len(message.MetaData) > 0 && string(message.MetaData) != "{}" {
		if err := json.Unmarshal(message.MetaData, &metaData); err != nil {
			return "", errors.NewInternalError("Failed to parse metadata")
		}
	}

	if metaData.Artifact == nil {
		return "", errors.NewNotFoundError("Artifact")
	}

	return metaData.Artifact.Content, nil
}

// MarkArtifactAsApplied marks an artifact as applied after it's been successfully applied
func (s *MessageService) MarkArtifactAsApplied(ctx context.Context, messageID uuid.UUID) error {
	message, err := s.GetMessageByID(ctx, messageID)
	if err != nil {
		return err
	}

	// Parse metadata
	var metaData metadata.MessageMetaData
	if len(message.MetaData) > 0 && string(message.MetaData) != "{}" {
		if err := json.Unmarshal(message.MetaData, &metaData); err != nil {
			return errors.NewInternalError("Failed to parse metadata")
		}
	}

	if metaData.Artifact == nil {
		return errors.NewNotFoundError("Artifact")
	}

	// Mark artifact as applied
	metaData.Artifact.Status = metadata.ArtifactStatusApplied
	now := time.Now()
	metaData.Artifact.AppliedAt = &now

	return s.UpdateMessageMetadata(ctx, messageID, &metaData)
}
