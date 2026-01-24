package middleware

import (
	"blog-agent-go/backend/internal/errors"
	"blog-agent-go/backend/internal/response"
	"blog-agent-go/backend/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AuthMiddleware validates JWT tokens from the Authorization header and sets the user ID in context.
// Returns a 401 Unauthorized error if the token is missing, invalid, or expired.
func AuthMiddleware(authService *services.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Get("Authorization")
		
		if token == "" {
			return response.Error(c, errors.NewUnauthorizedError("Not authenticated"))
		}

		if len(token) < 7 || token[:7] != "Bearer " {
			return response.Error(c, errors.NewUnauthorizedError("Invalid token format"))
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

