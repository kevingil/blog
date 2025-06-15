package services

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

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
	ID    int64  `json:"id"`
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
			ID:    int64(user.ID),
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
	err := db.QueryRow("SELECT id, name, email, passwordHash, role FROM users WHERE id = ?", userID).Scan(
		&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.Role,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("user not found")
		}
		return err
	}

	// Check if email is already taken by another user
	if req.Email != user.Email {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ? AND id != ?", req.Email, userID).Scan(&count)
		if err != nil {
			return err
		}
		if count > 0 {
			return errors.New("email already taken")
		}
	}

	// Update user
	_, err = db.Exec("UPDATE users SET name = ?, email = ? WHERE id = ?", req.Name, req.Email, userID)
	return err
}

func (s *AuthService) UpdatePassword(userID uint, req UpdatePasswordRequest) error {
	db := s.db.GetDB()

	// Get user
	var user models.User
	err := db.QueryRow("SELECT id, name, email, passwordHash, role FROM users WHERE id = ?", userID).Scan(
		&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.Role,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("user not found")
		}
		return err
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
	_, err = db.Exec("UPDATE users SET passwordHash = ? WHERE id = ?", string(hashedPassword), userID)
	return err
}

func (s *AuthService) DeleteAccount(userID uint, password string) error {
	db := s.db.GetDB()

	// Get user to verify password
	var user models.User
	err := db.QueryRow("SELECT id, name, email, passwordHash, role FROM users WHERE id = ?", userID).Scan(
		&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.Role,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("user not found")
		}
		return err
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return errors.New("password is incorrect")
	}

	// Delete user
	_, err = db.Exec("DELETE FROM users WHERE id = ?", userID)
	return err
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
		"iat":     time.Now().Unix(),
	})

	return token.SignedString(s.secretKey)
}

func (s *AuthService) VerifyToken(tokenString string) (*SessionData, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return s.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userData := claims["user"].(map[string]interface{})
		return &SessionData{
			User: UserData{
				ID: int64(userData["id"].(float64)),
			},
			Expires: claims["expires"].(string),
		}, nil
	}

	return nil, jwt.ErrInvalidKey
}

func (s *AuthService) SetSession(user *models.User) (*SessionData, error) {
	expiresInOneDay := time.Now().Add(24 * time.Hour)
	session := SessionData{
		User: UserData{
			ID: int64(user.ID),
		},
		Expires: expiresInOneDay.Format(time.RFC3339),
	}

	token, err := s.SignToken(session)
	if err != nil {
		return nil, err
	}

	// TODO: Set cookie in response
	_ = token

	return &session, nil
}
