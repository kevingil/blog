package models

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// ChatMessage represents a message in the article copilot chat
// Matches the chat_message table

type ChatMessage struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	ArticleID uuid.UUID      `json:"article_id" gorm:"type:uuid;not null;index"`
	Role      string         `json:"role" gorm:"not null"`
	Content   string         `json:"content" gorm:"type:text;not null"`
	MetaData  datatypes.JSON `json:"meta_data" gorm:"type:jsonb;default:'{}'"`
	CreatedAt string         `json:"created_at" gorm:"autoCreateTime"`
}

func (ChatMessage) TableName() string {
	return "chat_message"
}
