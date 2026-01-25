package auth

import (
	"context"
	"time"

	"backend/pkg/config"
	"backend/pkg/core"
	"backend/pkg/database"
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

// getRepo returns an account repository instance
func getRepo() *repository.AccountRepository {
	return repository.NewAccountRepository(database.DB())
}

// getSecretKey returns the JWT secret key from config
func getSecretKey() []byte {
	return []byte(config.Get().Auth.SecretKey)
}

// Login authenticates a user and returns a JWT token
func Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	repo := getRepo()

	account, err := repo.FindByEmail(ctx, req.Email)
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
func Register(ctx context.Context, req RegisterRequest) error {
	repo := getRepo()

	// Check if email already exists
	existing, err := repo.FindByEmail(ctx, req.Email)
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

	return repo.Save(ctx, account)
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

// UpdateAccount updates account information
func UpdateAccount(ctx context.Context, accountID uuid.UUID, req UpdateAccountRequest) error {
	repo := getRepo()

	account, err := repo.FindByID(ctx, accountID)
	if err != nil {
		return err
	}

	// Check if email is already taken by another account
	if req.Email != account.Email {
		existing, err := repo.FindByEmail(ctx, req.Email)
		if err != nil && err != core.ErrNotFound {
			return err
		}
		if existing != nil && existing.ID != accountID {
			return core.ErrAlreadyExists
		}
	}

	account.Name = req.Name
	account.Email = req.Email

	return repo.Update(ctx, account)
}

// UpdatePassword changes the user's password
func UpdatePassword(ctx context.Context, accountID uuid.UUID, req UpdatePasswordRequest) error {
	repo := getRepo()

	account, err := repo.FindByID(ctx, accountID)
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
	return repo.Update(ctx, account)
}

// DeleteAccount removes a user account after verifying password
func DeleteAccount(ctx context.Context, accountID uuid.UUID, password string) error {
	repo := getRepo()

	account, err := repo.FindByID(ctx, accountID)
	if err != nil {
		return err
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password)); err != nil {
		return core.ErrUnauthorized
	}

	return repo.Delete(ctx, accountID)
}

// GetAccount retrieves an account by ID
func GetAccount(ctx context.Context, accountID uuid.UUID) (*types.Account, error) {
	repo := getRepo()
	return repo.FindByID(ctx, accountID)
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

// Legacy Service type for backward compatibility during migration

// Service provides business logic for authentication (deprecated - use package functions)
type Service struct {
	store     AccountStore
	secretKey []byte
}

// NewService creates a new auth service (deprecated - use package functions)
func NewService(store AccountStore, secretKey string) *Service {
	return &Service{
		store:     store,
		secretKey: []byte(secretKey),
	}
}

// Legacy method wrappers for backward compatibility

func (s *Service) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	return Login(ctx, req)
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) error {
	return Register(ctx, req)
}

func (s *Service) ValidateToken(tokenString string) (*jwt.Token, error) {
	return ValidateToken(tokenString)
}

func (s *Service) GetUserIDFromToken(token *jwt.Token) (uuid.UUID, error) {
	return GetUserIDFromToken(token)
}

func (s *Service) UpdateAccount(ctx context.Context, accountID uuid.UUID, req UpdateAccountRequest) error {
	return UpdateAccount(ctx, accountID, req)
}

func (s *Service) UpdatePassword(ctx context.Context, accountID uuid.UUID, req UpdatePasswordRequest) error {
	return UpdatePassword(ctx, accountID, req)
}

func (s *Service) DeleteAccount(ctx context.Context, accountID uuid.UUID, password string) error {
	return DeleteAccount(ctx, accountID, password)
}

func (s *Service) GetAccount(ctx context.Context, accountID uuid.UUID) (*types.Account, error) {
	return GetAccount(ctx, accountID)
}
