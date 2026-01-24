package auth

import (
	"context"
	"time"

	"blog-agent-go/backend/internal/core"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Service provides business logic for authentication
type Service struct {
	store     AccountStore
	secretKey []byte
}

// NewService creates a new auth service
func NewService(store AccountStore, secretKey string) *Service {
	return &Service{
		store:     store,
		secretKey: []byte(secretKey),
	}
}

// LoginRequest represents login credentials
type LoginRequest struct {
	Email    string
	Password string
}

// LoginResponse represents the result of a successful login
type LoginResponse struct {
	Token string
	User  UserData
}

// UserData represents user information returned after authentication
type UserData struct {
	ID    string
	Name  string
	Email string
	Role  string
}

// Login authenticates a user and returns a JWT token
func (s *Service) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	account, err := s.store.FindByEmail(ctx, req.Email)
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

	tokenString, err := token.SignedString(s.secretKey)
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
	existing, err := s.store.FindByEmail(ctx, req.Email)
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

	account := &Account{
		ID:           uuid.New(),
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Role:         "user",
	}

	return s.store.Save(ctx, account)
}

// ValidateToken validates a JWT token and returns the parsed token
func (s *Service) ValidateToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, core.ErrUnauthorized
		}
		return s.secretKey, nil
	})

	if err != nil || !token.Valid {
		return nil, core.ErrUnauthorized
	}

	return token, nil
}

// GetUserIDFromToken extracts the user ID from a validated token
func (s *Service) GetUserIDFromToken(token *jwt.Token) (uuid.UUID, error) {
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

// UpdateAccountRequest represents a request to update account info
type UpdateAccountRequest struct {
	Name  string
	Email string
}

// UpdateAccount updates account information
func (s *Service) UpdateAccount(ctx context.Context, accountID uuid.UUID, req UpdateAccountRequest) error {
	account, err := s.store.FindByID(ctx, accountID)
	if err != nil {
		return err
	}

	// Check if email is already taken by another account
	if req.Email != account.Email {
		existing, err := s.store.FindByEmail(ctx, req.Email)
		if err != nil && err != core.ErrNotFound {
			return err
		}
		if existing != nil && existing.ID != accountID {
			return core.ErrAlreadyExists
		}
	}

	account.Name = req.Name
	account.Email = req.Email

	return s.store.Update(ctx, account)
}

// UpdatePasswordRequest represents a request to change password
type UpdatePasswordRequest struct {
	CurrentPassword string
	NewPassword     string
}

// UpdatePassword changes the user's password
func (s *Service) UpdatePassword(ctx context.Context, accountID uuid.UUID, req UpdatePasswordRequest) error {
	account, err := s.store.FindByID(ctx, accountID)
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
	return s.store.Update(ctx, account)
}

// DeleteAccount removes a user account after verifying password
func (s *Service) DeleteAccount(ctx context.Context, accountID uuid.UUID, password string) error {
	account, err := s.store.FindByID(ctx, accountID)
	if err != nil {
		return err
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password)); err != nil {
		return core.ErrUnauthorized
	}

	return s.store.Delete(ctx, accountID)
}

// GetAccount retrieves an account by ID
func (s *Service) GetAccount(ctx context.Context, accountID uuid.UUID) (*Account, error) {
	return s.store.FindByID(ctx, accountID)
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
