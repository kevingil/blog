// Package middleware provides HTTP middleware functions for the application
package middleware

import (
	"errors"
	"fmt"

	"backend/pkg/database"
	"backend/pkg/database/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Context key constants
const (
	UserIDKey = "userID"
	OrgIDKey  = "orgID"
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

// GetOrgID retrieves the organization ID for the authenticated user.
// Returns nil if the user is not associated with an organization.
func GetOrgID(c *fiber.Ctx) *uuid.UUID {
	// First check if already cached in context
	orgIDValue := c.Locals(OrgIDKey)
	if orgIDValue != nil {
		if orgID, ok := orgIDValue.(*uuid.UUID); ok {
			return orgID
		}
	}

	// Fetch from database
	userID, err := GetUserID(c)
	if err != nil {
		return nil
	}

	repo := repository.NewAccountRepository(database.DB())
	account, err := repo.FindByID(c.Context(), userID)
	if err != nil {
		return nil
	}

	// Cache in context for subsequent calls
	c.Locals(OrgIDKey, account.OrganizationID)
	return account.OrganizationID
}

// SetOrgID stores the organization ID in the fiber context for later retrieval.
func SetOrgID(c *fiber.Ctx, orgID *uuid.UUID) {
	c.Locals(OrgIDKey, orgID)
}

