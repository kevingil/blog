package auth

import (
	"context"
	"time"

	"backend/pkg/config"
	"backend/pkg/core"
	"backend/pkg/database/repository"
	"backend/pkg/types"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// LoginRequest represents login credentials
type LoginRequest = types.LoginRequest

// LoginResponse represents the result of a successful login
type LoginResponse = types.LoginResponse

// UserData represents user information returned after authentication
type UserData = types.UserData

// UpdateAccountRequest represents a request to update account info
type UpdateAccountRequest = types.UpdateAccountRequest

// UpdatePasswordRequest represents a request to change password
type UpdatePasswordRequest = types.UpdatePasswordRequest

// Service provides business logic for authentication
type Service struct {
	repo repository.AccountRepository
}

// NewService creates a new auth service with the provided repository
func NewService(repo repository.AccountRepository) *Service {
	return &Service{
		repo: repo,
	}
}

// getSecretKey returns the JWT secret key from config
func getSecretKey() []byte {
	return []byte(config.Get().Auth.SecretKey)
}

// Login authenticates a user and returns a JWT token
func (s *Service) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	account, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, core.ErrUnauthorized
	}

	if err := bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(req.Password)); err != nil {
		return nil, core.ErrUnauthorized
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": account.ID,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString(getSecretKey())
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token: tokenString,
		User: UserData{
			ID:    account.ID.String(),
			Name:  account.Name,
			Email: account.Email,
			Role:  account.Role,
		},
	}, nil
}

// Register creates a new user account
func (s *Service) Register(ctx context.Context, req RegisterRequest) error {
	// Check if email already exists
	existing, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil && err != core.ErrNotFound {
		return err
	}
	if existing != nil {
		return core.ErrAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	account := &types.Account{
		ID:           uuid.New(),
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Role:         "user",
	}

	return s.repo.Save(ctx, account)
}

// UpdateAccount updates account information
func (s *Service) UpdateAccount(ctx context.Context, accountID uuid.UUID, req UpdateAccountRequest) error {
	account, err := s.repo.FindByID(ctx, accountID)
	if err != nil {
		return err
	}

	// Check if email is already taken by another account
	if req.Email != account.Email {
		existing, err := s.repo.FindByEmail(ctx, req.Email)
		if err != nil && err != core.ErrNotFound {
			return err
		}
		if existing != nil && existing.ID != accountID {
			return core.ErrAlreadyExists
		}
	}

	account.Name = req.Name
	account.Email = req.Email

	return s.repo.Update(ctx, account)
}

// UpdatePassword changes the user's password
func (s *Service) UpdatePassword(ctx context.Context, accountID uuid.UUID, req UpdatePasswordRequest) error {
	account, err := s.repo.FindByID(ctx, accountID)
	if err != nil {
		return err
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return core.ErrUnauthorized
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	account.PasswordHash = string(hashedPassword)
	return s.repo.Update(ctx, account)
}

// DeleteAccount removes a user account after verifying password
func (s *Service) DeleteAccount(ctx context.Context, accountID uuid.UUID, password string) error {
	account, err := s.repo.FindByID(ctx, accountID)
	if err != nil {
		return err
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password)); err != nil {
		return core.ErrUnauthorized
	}

	return s.repo.Delete(ctx, accountID)
}

// GetAccount retrieves an account by ID
func (s *Service) GetAccount(ctx context.Context, accountID uuid.UUID) (*types.Account, error) {
	return s.repo.FindByID(ctx, accountID)
}

// ValidateToken validates a JWT token and returns the parsed token
func ValidateToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, core.ErrUnauthorized
		}
		return getSecretKey(), nil
	})

	if err != nil || !token.Valid {
		return nil, core.ErrUnauthorized
	}

	return token, nil
}

// GetUserIDFromToken extracts the user ID from a validated token
func GetUserIDFromToken(token *jwt.Token) (uuid.UUID, error) {
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, core.ErrUnauthorized
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return uuid.Nil, core.ErrUnauthorized
	}

	return uuid.Parse(sub)
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// ComparePasswords compares a plain text password with a hashed password
func ComparePasswords(plainTextPassword, hashedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainTextPassword))
	return err == nil
}
