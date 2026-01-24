// Package auth provides the Account domain type and authentication logic
package auth

import (
	"context"
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

// AccountStore defines the data access interface for accounts
type AccountStore interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Account, error)
	FindByEmail(ctx context.Context, email string) (*Account, error)
	Save(ctx context.Context, account *Account) error
	Update(ctx context.Context, account *Account) error
	Delete(ctx context.Context, id uuid.UUID) error
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
