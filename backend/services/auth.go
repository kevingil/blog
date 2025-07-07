package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"blog-agent-go/backend/database"
	"blog-agent-go/backend/models"
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
	Password string `json:"password" validate:"required,min=8"`
}

type LoginResponse struct {
	Token string   `json:"token"`
	User  UserData `json:"user"`
}

type RegisterRequest struct {
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type UpdateAccountRequest struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
}

type UpdatePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" validate:"required,min=8"`
	NewPassword     string `json:"newPassword" validate:"required,min=8"`
}

type SessionData struct {
	User    UserData `json:"user"`
	Expires string   `json:"expires"`
}

type UserData struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (s *AuthService) Login(req LoginRequest) (*LoginResponse, error) {
	fmt.Printf("Login request received for email:'%s'\n", req.Email)
	user, err := s.db.GetUserByEmail(req.Email)
	if err != nil {
		return nil, errors.New("invalid account credentials")
	}
	if user == nil {
		return nil, errors.New("invalid account credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid user credentials")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString(s.secretKey)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token: tokenString,
		User: UserData{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
			Role:  user.Role,
		},
	}, nil
}

func (s *AuthService) Register(req RegisterRequest) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := models.User{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Role:         "user",
	}

	return s.db.CreateUser(&user)
}

func (s *AuthService) ValidateToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return token, nil
}

func (s *AuthService) UpdateAccount(userID uint, req UpdateAccountRequest) error {
	db := s.db.GetDB()

	// Check if user exists
	var user models.User
	result := db.First(&user, userID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return errors.New("user not found")
		}
		return result.Error
	}

	// Check if email is already taken by another user
	if req.Email != user.Email {
		var count int64
		db.Model(&models.User{}).Where("email = ? AND id != ?", req.Email, userID).Count(&count)
		if count > 0 {
			return errors.New("email already taken")
		}
	}

	// Update user
	result = db.Model(&user).Updates(models.User{
		Name:  req.Name,
		Email: req.Email,
	})
	return result.Error
}

func (s *AuthService) UpdatePassword(userID uint, req UpdatePasswordRequest) error {
	db := s.db.GetDB()

	// Get user
	var user models.User
	result := db.First(&user, userID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return errors.New("user not found")
		}
		return result.Error
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return errors.New("current password is incorrect")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Update password
	result = db.Model(&user).Update("password_hash", string(hashedPassword))
	return result.Error
}

func (s *AuthService) DeleteAccount(userID uint, password string) error {
	db := s.db.GetDB()

	// Get user
	var user models.User
	result := db.First(&user, userID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return errors.New("user not found")
		}
		return result.Error
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return errors.New("password is incorrect")
	}

	// Delete user
	result = db.Delete(&user)
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

func (s *AuthService) SignToken(payload SessionData) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user":    payload.User,
		"expires": payload.Expires,
	})

	return token.SignedString(s.secretKey)
}

func (s *AuthService) VerifyToken(tokenString string) (*SessionData, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	// Extract user data from claims
	// Note: This implementation may need adjustment based on the exact structure of your JWT claims
	return &SessionData{
		Expires: claims["expires"].(string),
	}, nil
}

func (s *AuthService) SetSession(user *models.User) (*SessionData, error) {
	sessionData := &SessionData{
		User: UserData{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
			Role:  user.Role,
		},
		Expires: time.Now().Add(time.Hour * 24).Format(time.RFC3339),
	}

	return sessionData, nil
}
