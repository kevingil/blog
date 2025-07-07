package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Name         string `json:"name" gorm:"not null"`
	Email        string `json:"email" gorm:"uniqueIndex;not null"`
	PasswordHash string `json:"-" gorm:"not null"`
	Role         string `json:"role" gorm:"default:user;not null"`
}
