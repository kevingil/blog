// Package auth provides authentication logic
package auth

import (
	"context"

	"backend/pkg/types"

	"github.com/google/uuid"
)

// AccountStore defines the data access interface for accounts
type AccountStore interface {
	FindByID(ctx context.Context, id uuid.UUID) (*types.Account, error)
	FindByEmail(ctx context.Context, email string) (*types.Account, error)
	Save(ctx context.Context, account *types.Account) error
	Update(ctx context.Context, account *types.Account) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// Account is an alias to types.Account for backward compatibility
type Account = types.Account

// Credentials is an alias to types.Credentials for backward compatibility
type Credentials = types.Credentials

// RegisterRequest is an alias to types.RegisterRequest for backward compatibility
type RegisterRequest = types.RegisterRequest
