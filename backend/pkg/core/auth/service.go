package auth

import (
	"context"
	"time"

	"backend/pkg/config"
	"backend/pkg/core"
	"backend/pkg/database"
	"backend/pkg/database/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

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

// getSecretKey returns the JWT secret key from config
func getSecretKey() []byte {
	return []byte(config.Get().Auth.SecretKey)
}

// Login authenticates a user and returns a JWT token
func Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	db := database.DB()

	var model models.Account
	if err := db.WithContext(ctx).Where("email = ?", req.Email).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrUnauthorized
		}
		return nil, core.ErrUnauthorized
	}

	if err := bcrypt.CompareHashAndPassword([]byte(model.PasswordHash), []byte(req.Password)); err != nil {
		return nil, core.ErrUnauthorized
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": model.ID,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString(getSecretKey())
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token: tokenString,
		User: UserData{
			ID:    model.ID.String(),
			Name:  model.Name,
			Email: model.Email,
			Role:  model.Role,
		},
	}, nil
}

// Register creates a new user account
func Register(ctx context.Context, req RegisterRequest) error {
	db := database.DB()

	// Check if email already exists
	var count int64
	db.WithContext(ctx).Model(&models.Account{}).Where("email = ?", req.Email).Count(&count)
	if count > 0 {
		return core.ErrAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	model := &models.Account{
		ID:           uuid.New(),
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Role:         "user",
	}

	return db.WithContext(ctx).Create(model).Error
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
	db := database.DB()

	var model models.Account
	if err := db.WithContext(ctx).First(&model, accountID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return core.ErrNotFound
		}
		return err
	}

	// Check if email is already taken by another account
	if req.Email != model.Email {
		var count int64
		db.WithContext(ctx).Model(&models.Account{}).Where("email = ? AND id != ?", req.Email, accountID).Count(&count)
		if count > 0 {
			return core.ErrAlreadyExists
		}
	}

	model.Name = req.Name
	model.Email = req.Email

	return db.WithContext(ctx).Save(&model).Error
}

// UpdatePassword changes the user's password
func UpdatePassword(ctx context.Context, accountID uuid.UUID, req UpdatePasswordRequest) error {
	db := database.DB()

	var model models.Account
	if err := db.WithContext(ctx).First(&model, accountID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return core.ErrNotFound
		}
		return err
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(model.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return core.ErrUnauthorized
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	model.PasswordHash = string(hashedPassword)
	return db.WithContext(ctx).Save(&model).Error
}

// DeleteAccount removes a user account after verifying password
func DeleteAccount(ctx context.Context, accountID uuid.UUID, password string) error {
	db := database.DB()

	var model models.Account
	if err := db.WithContext(ctx).First(&model, accountID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return core.ErrNotFound
		}
		return err
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(model.PasswordHash), []byte(password)); err != nil {
		return core.ErrUnauthorized
	}

	return db.WithContext(ctx).Delete(&model).Error
}

// GetAccount retrieves an account by ID
func GetAccount(ctx context.Context, accountID uuid.UUID) (*Account, error) {
	db := database.DB()

	var model models.Account
	if err := db.WithContext(ctx).First(&model, accountID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}

	// Manual conversion since models no longer import core
	return &Account{
		ID:             model.ID,
		Name:           model.Name,
		Email:          model.Email,
		PasswordHash:   model.PasswordHash,
		Role:           model.Role,
		CreatedAt:      model.CreatedAt,
		UpdatedAt:      model.UpdatedAt,
		OrganizationID: model.OrganizationID,
	}, nil
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

func (s *Service) GetAccount(ctx context.Context, accountID uuid.UUID) (*Account, error) {
	return GetAccount(ctx, accountID)
}
