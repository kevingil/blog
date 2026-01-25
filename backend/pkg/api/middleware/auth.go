package middleware

import (
	"backend/pkg/api/response"
	"backend/pkg/api/services"
	"backend/pkg/config"
	"backend/pkg/core"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Auth returns a middleware that validates JWT tokens from the Authorization header.
// It pulls the secret key from config automatically.
// Returns a 401 Unauthorized error if the token is missing, invalid, or expired.
func Auth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenHeader := c.Get("Authorization")

		if tokenHeader == "" {
			return response.Error(c, core.UnauthorizedError("Not authenticated"))
		}

		if len(tokenHeader) < 7 || tokenHeader[:7] != "Bearer " {
			return response.Error(c, core.UnauthorizedError("Invalid token format"))
		}

		tokenString := tokenHeader[7:]
		secretKey := []byte(config.Get().Auth.SecretKey)

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, core.UnauthorizedError("Unexpected signing method")
			}
			return secretKey, nil
		})

		if err != nil || !token.Valid {
			return response.Error(c, core.UnauthorizedError("Invalid or expired token"))
		}

		claims := token.Claims.(jwt.MapClaims)
		userID := uuid.MustParse(claims["sub"].(string))

		SetUserID(c, userID)
		return c.Next()
	}
}

// AuthMiddleware validates JWT tokens from the Authorization header and sets the user ID in context.
// Returns a 401 Unauthorized error if the token is missing, invalid, or expired.
// Deprecated: Use Auth() instead which pulls dependencies automatically.
func AuthMiddleware(authService *services.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Get("Authorization")

		if token == "" {
			return response.Error(c, core.UnauthorizedError("Not authenticated"))
		}

		if len(token) < 7 || token[:7] != "Bearer " {
			return response.Error(c, core.UnauthorizedError("Invalid token format"))
		}

		token = token[7:]

		validToken, err := authService.ValidateToken(token)
		if err != nil {
			return response.Error(c, err)
		}

		claims := validToken.Claims.(jwt.MapClaims)
		userID := uuid.MustParse(claims["sub"].(string))

		SetUserID(c, userID)
		return c.Next()
	}
}
