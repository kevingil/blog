package services

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"backend/pkg/core"
	"backend/pkg/database"
	"backend/pkg/database/models"

	"github.com/google/uuid"
)

type AuthService struct {
	db        database.Service
	secretKey []byte
}

func NewAuthService(db database.Service, secretKey string) *AuthService {
	return &AuthService{
		db:        db,
		secretKey: []byte(secretKey),
	}
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type LoginResponse struct {
	Token string   `json:"token"`
	User  UserData `json:"user"`
}

type RegisterRequest struct {
	Name     string `json:"name" validate:"required,min=2,max=100"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type UpdateAccountRequest struct {
	Name  string `json:"name" validate:"required,min=2,max=100"`
	Email string `json:"email" validate:"required,email"`
}

type UpdatePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" validate:"required,min=6"`
	NewPassword     string `json:"newPassword" validate:"required,min=6"`
}

type SessionData struct {
	User    UserData `json:"user"`
	Expires string   `json:"expires"`
}

type UserData struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (s *AuthService) Login(req LoginRequest) (*LoginResponse, error) {
	account, err := s.db.GetAccountByEmail(req.Email)
	if err != nil {
		return nil, core.UnauthorizedError("Invalid credentials")
	}
	if account == nil {
		return nil, core.UnauthorizedError("Invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(req.Password)); err != nil {
		return nil, core.UnauthorizedError("Invalid credentials")
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

func (s *AuthService) Register(req RegisterRequest) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	account := models.Account{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Role:         "user",
	}

	return s.db.CreateAccount(&account)
}

func (s *AuthService) ValidateToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, core.InvalidInputError("Unexpected signing method")
		}
		return s.secretKey, nil
	})

	if err != nil {
		return nil, core.UnauthorizedError("Invalid or expired token")
	}

	if !token.Valid {
		return nil, core.UnauthorizedError("Invalid token")
	}

	return token, nil
}

func (s *AuthService) UpdateAccount(accountID uuid.UUID, req UpdateAccountRequest) error {
	db := s.db.GetDB()

	// Check if account exists
	var account models.Account
	result := db.First(&account, accountID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return core.NotFoundError("Account")
		}
		return core.InternalError("Failed to fetch account")
	}

	// Check if email is already taken by another account
	if req.Email != account.Email {
		var count int64
		db.Model(&models.Account{}).Where("email = ? AND id != ?", req.Email, accountID).Count(&count)
		if count > 0 {
			return core.AlreadyExistsError("Email")
		}
	}

	// Update account
	result = db.Model(&account).Updates(models.Account{
		Name:  req.Name,
		Email: req.Email,
	})
	return result.Error
}

func (s *AuthService) UpdatePassword(accountID uuid.UUID, req UpdatePasswordRequest) error {
	db := s.db.GetDB()

	// Get account
	var account models.Account
	result := db.First(&account, accountID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return core.NotFoundError("Account")
		}
		return core.InternalError("Failed to fetch account")
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return core.UnauthorizedError("Current password is incorrect")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Update password
	result = db.Model(&account).Update("password_hash", string(hashedPassword))
	return result.Error
}

func (s *AuthService) DeleteAccount(accountID uuid.UUID, password string) error {
	db := s.db.GetDB()

	// Get account
	var account models.Account
	result := db.First(&account, accountID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return core.NotFoundError("Account")
		}
		return core.InternalError("Failed to fetch account")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password)); err != nil {
		return core.UnauthorizedError("Password is incorrect")
	}

	// Delete account
	result = db.Delete(&account)
	return result.Error
}

func (s *AuthService) HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func (s *AuthService) ComparePasswords(plainTextPassword string, hashedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainTextPassword))
	return err == nil
}
