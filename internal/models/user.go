package models

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Name         string `json:"name" gorm:"type:varchar(255);not null"`
	Email        string `json:"email" gorm:"type:varchar(255);not null;unique"`
	PasswordHash string `json:"-" gorm:"type:varchar(255);not null"`
	Role         string `json:"role" gorm:"type:varchar(50);not null;default:'user'"`
}
