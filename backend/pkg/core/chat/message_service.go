// Package chat provides chat message persistence and management
package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"backend/pkg/core"
	"backend/pkg/core/agent/metadata"
	"backend/pkg/database"
	"backend/pkg/database/models"
	"backend/pkg/database/repository"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// MessageService provides operations for managing chat messages
type MessageService struct {
	repo *repository.ChatMessageRepository
}

// NewMessageService creates a new chat message service
func NewMessageService(db database.Service) *MessageService {
	return &MessageService{
		repo: repository.NewChatMessageRepository(db.GetDB()),
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
			return nil, core.InvalidInputError(fmt.Sprintf("Invalid metadata: %v", err))
		}

		jsonBytes, err := json.Marshal(metaData)
		if err != nil {
			log.Printf("[MessageService] ❌ Failed to marshal metadata: %v", err)
			return nil, core.InternalError("Failed to marshal metadata")
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

	if err := s.repo.Create(ctx, message); err != nil {
		log.Printf("[MessageService] ❌ Database INSERT failed: %v", err)
		return nil, core.InternalError("Failed to save message")
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

	messages, err := s.repo.GetByArticleID(ctx, articleID, limit)
	if err != nil {
		fmt.Printf("[MessageService] ❌ Database query failed: %v\n", err)
		return nil, core.InternalError("Failed to retrieve conversation history")
	}

	// Reverse to get chronological order (oldest first for display)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	fmt.Printf("[MessageService] ✅ Fetched %d messages\n", len(messages))

	// If no messages exist, create and save a default initial greeting
	if len(messages) == 0 {
		fmt.Printf("[MessageService] 📝 No messages found, creating initial greeting\n")

		initialMessage := &models.ChatMessage{
			ArticleID: articleID,
			Role:      "assistant",
			Content:   initialGreeting,
			MetaData:  datatypes.JSON("{}"),
		}

		// Save to database so it persists
		if err := s.repo.Create(ctx, initialMessage); err != nil {
			fmt.Printf("[MessageService] ⚠️  Failed to save initial greeting: %v\n", err)
			// Still return it even if save failed
			initialMessage.ID = uuid.New()
			initialMessage.CreatedAt = time.Now()
		} else {
			fmt.Printf("[MessageService] ✅ Saved initial greeting to database (ID: %s)\n", initialMessage.ID)
		}

		return []models.ChatMessage{*initialMessage}, nil
	}

	// Log each message from database
	for _, msg := range messages {
		preview := msg.Content
		if len(preview) > 50 {
			preview = preview[:50] + "..."
		}
		// fmt.Printf("[MessageService]    [%d] %s (ID: %s): %s\n", i+1, msg.Role, msg.ID, preview)

		// Check if metadata is present
		metadataSize := len(msg.MetaData)
		if metadataSize > 2 {
			// fmt.Printf("[MessageService]        Metadata: %d bytes\n", metadataSize)
		}
	}

	return messages, nil
}

// ClearConversationHistory deletes all chat messages for an article
func (s *MessageService) ClearConversationHistory(ctx context.Context, articleID uuid.UUID) error {
	log.Printf("[MessageService] 🗑️ ClearConversationHistory called for article: %s", articleID)

	rowsAffected, err := s.repo.DeleteByArticleID(ctx, articleID)
	if err != nil {
		log.Printf("[MessageService] ❌ Failed to clear conversation history: %v", err)
		return core.InternalError("Failed to clear conversation history")
	}

	log.Printf("[MessageService] ✅ Cleared %d messages for article %s", rowsAffected, articleID)
	return nil
}

// UpdateMessageMetadata updates the metadata of an existing message
func (s *MessageService) UpdateMessageMetadata(ctx context.Context, messageID uuid.UUID, metaData *metadata.MessageMetaData) error {
	// Validate metadata
	if err := metadata.ValidateMetaData(metaData); err != nil {
		return core.InvalidInputError(fmt.Sprintf("Invalid metadata: %v", err))
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(metaData)
	if err != nil {
		return core.InternalError("Failed to marshal metadata")
	}

	rowsAffected, err := s.repo.UpdateMetaData(ctx, messageID, jsonBytes)
	if err != nil {
		return core.InternalError("Failed to update message metadata")
	}

	if rowsAffected == 0 {
		return core.NotFoundError("Message")
	}

	return nil
}

// UpdateArtifactStatus updates the status of an artifact within a message
func (s *MessageService) UpdateArtifactStatus(ctx context.Context, messageID uuid.UUID, artifactID, status string) error {
	// Get the current message
	message, err := s.repo.GetByID(ctx, messageID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return core.NotFoundError("Message")
		}
		return core.InternalError("Failed to retrieve message")
	}

	// Parse current metadata
	var metaData metadata.MessageMetaData
	if len(message.MetaData) > 0 && string(message.MetaData) != "{}" {
		if err := json.Unmarshal(message.MetaData, &metaData); err != nil {
			return core.InternalError("Failed to parse message metadata")
		}
	}

	// Update artifact status
	if metaData.Artifact == nil {
		return core.NotFoundError("Artifact")
	}

	if artifactID != "" && metaData.Artifact.ID != artifactID {
		return core.NotFoundError("Artifact")
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
	message, err := s.repo.GetByID(ctx, messageID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.NotFoundError("Message")
		}
		return nil, core.InternalError("Failed to retrieve message")
	}

	return message, nil
}

// GetPendingArtifacts retrieves all messages with pending artifacts for an article
func (s *MessageService) GetPendingArtifacts(ctx context.Context, articleID uuid.UUID) ([]models.ChatMessage, error) {
	messages, err := s.repo.GetPendingArtifacts(ctx, articleID)
	if err != nil {
		return nil, core.InternalError("Failed to retrieve pending artifacts")
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
			return core.InternalError("Failed to parse metadata")
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
			return core.InternalError("Failed to parse metadata")
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
			return "", core.InternalError("Failed to parse metadata")
		}
	}

	if metaData.Artifact == nil {
		return "", core.NotFoundError("Artifact")
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
			return core.InternalError("Failed to parse metadata")
		}
	}

	if metaData.Artifact == nil {
		return core.NotFoundError("Artifact")
	}

	// Mark artifact as applied
	metaData.Artifact.Status = metadata.ArtifactStatusApplied
	now := time.Now()
	metaData.Artifact.AppliedAt = &now

	return s.UpdateMessageMetadata(ctx, messageID, &metaData)
}
