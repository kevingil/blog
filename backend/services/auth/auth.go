package auth

import (
	"time"

	"blog-agent-go/backend/models"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	secretKey []byte
}

func NewAuthService(secretKey string) *AuthService {
	return &AuthService{
		secretKey: []byte(secretKey),
	}
}

type SessionData struct {
	User    UserData `json:"user"`
	Expires string   `json:"expires"`
}

type UserData struct {
	ID int64 `json:"id"`
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
