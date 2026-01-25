package profile

import (
	"backend/pkg/api/middleware"
	"backend/pkg/api/response"
	"backend/pkg/core"
	coreProfile "backend/pkg/core/profile"

	"github.com/gofiber/fiber/v2"
)

// GetPublicProfile handles GET /profile/public
func GetPublicProfile(c *fiber.Ctx) error {
	profile, err := coreProfile.GetPublicProfile(c.Context())
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, profile)
}

// GetMyProfile handles GET /profile
func GetMyProfile(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return response.Error(c, core.UnauthorizedError("Not authenticated"))
	}

	profile, err := coreProfile.GetUserProfile(c.Context(), userID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, profile)
}

// UpdateProfile handles PUT /profile
func UpdateProfile(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return response.Error(c, core.UnauthorizedError("Not authenticated"))
	}

	var req coreProfile.ProfileUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}

	profile, err := coreProfile.UpdateUserProfile(c.Context(), userID, req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, profile)
}

// GetSiteSettings handles GET /profile/settings
func GetSiteSettings(c *fiber.Ctx) error {
	settings, err := coreProfile.GetSiteSettings(c.Context())
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, settings)
}

// UpdateSiteSettings handles PUT /profile/settings
func UpdateSiteSettings(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return response.Error(c, core.UnauthorizedError("Not authenticated"))
	}

	// Check if user is admin
	isAdmin, err := coreProfile.IsUserAdmin(c.Context(), userID)
	if err != nil {
		return response.Error(c, err)
	}
	if !isAdmin {
		return response.Error(c, core.ForbiddenError("Only admins can update site settings"))
	}

	var req coreProfile.SiteSettingsUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}

	settings, err := coreProfile.UpdateSiteSettings(c.Context(), req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, settings)
}
