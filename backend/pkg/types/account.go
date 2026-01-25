// Package types provides shared domain types used across the application.
// This package is a "leaf" package that should have no internal imports
// to prevent import cycles between repository and core packages.
package types

import (
	"time"

	"github.com/google/uuid"
)

// Account represents a user account
type Account struct {
	ID           uuid.UUID
	Name         string
	Email        string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
	UpdatedAt    time.Time

	// Profile fields
	Bio             *string
	ProfileImage    *string
	EmailPublic     *string
	SocialLinks     map[string]interface{}
	MetaDescription *string

	// Organization relationship (optional)
	OrganizationID *uuid.UUID
}

// Credentials represents login credentials
type Credentials struct {
	Email    string
	Password string
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Name     string
	Email    string
	Password string
}

// LoginRequest represents login credentials
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

// LoginResponse represents the result of a successful login
type LoginResponse struct {
	Token string   `json:"token"`
	User  UserData `json:"user"`
}

// UserData represents user information returned after authentication
type UserData struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// UpdateAccountRequest represents a request to update account info
type UpdateAccountRequest struct {
	Name  string `json:"name" validate:"required,min=2,max=100"`
	Email string `json:"email" validate:"required,email"`
}

// UpdatePasswordRequest represents a request to change password
type UpdatePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" validate:"required,min=6"`
	NewPassword     string `json:"newPassword" validate:"required,min=6"`
}
