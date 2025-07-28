package models

import "github.com/google/uuid"

type Account struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Name         string    `json:"name" gorm:"not null"`
	Email        string    `json:"email" gorm:"uniqueIndex;not null"`
	PasswordHash string    `json:"-" gorm:"not null;column:password_hash"`
	Role         string    `json:"role" gorm:"default:user;not null"`
	CreatedAt    string    `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    string    `json:"updated_at" gorm:"autoUpdateTime"`
}

func (Account) TableName() string {
	return "account"
}
