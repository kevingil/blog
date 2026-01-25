package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// ChatMessage is the GORM model for chat messages
// Note: This stays close to the database as it's used by core/chat
type ChatMessage struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	ArticleID uuid.UUID      `json:"article_id" gorm:"type:uuid;not null;index"`
	Role      string         `json:"role" gorm:"not null"`
	Content   string         `json:"content" gorm:"type:text;not null"`
	MetaData  datatypes.JSON `json:"meta_data" gorm:"type:jsonb;default:'{}'"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
}

func (ChatMessage) TableName() string {
	return "chat_message"
}
