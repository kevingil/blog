// Package middleware provides HTTP middleware functions for the application
package middleware

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Context key constants
const (
	UserIDKey = "userID"
)

// GetUserID safely extracts the user ID from the fiber context.
// Returns an error if the user ID is not found or cannot be converted to uuid.UUID.
func GetUserID(c *fiber.Ctx) (uuid.UUID, error) {
	userIDValue := c.Locals(UserIDKey)
	if userIDValue == nil {
		return uuid.UUID{}, errors.New("user ID not found in context")
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		return uuid.UUID{}, fmt.Errorf("user ID has invalid type: %T", userIDValue)
	}

	return userID, nil
}

// SetUserID stores the user ID in the fiber context for later retrieval.
func SetUserID(c *fiber.Ctx, userID uuid.UUID) {
	c.Locals(UserIDKey, userID)
}

