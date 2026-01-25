package profile

import (
	"backend/pkg/api/middleware"
	"backend/pkg/api/response"
	"backend/pkg/api/services"
	"backend/pkg/core"

	"github.com/gofiber/fiber/v2"
)

// GetPublicProfile handles GET /profile/public
func GetPublicProfile(c *fiber.Ctx) error {
	profile, err := services.Profiles().GetPublicProfile()
	if err != nil {
		return response.Error(c, err)
	}
	if profile == nil {
		return response.Error(c, core.NotFoundError("Public profile"))
	}
	return response.Success(c, profile)
}

// GetMyProfile handles GET /profile
func GetMyProfile(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return response.Error(c, core.UnauthorizedError("Not authenticated"))
	}

	profile, err := services.Profiles().GetUserProfile(userID)
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

	var req services.ProfileUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}

	profile, err := services.Profiles().UpdateUserProfile(userID, req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, profile)
}

// GetSiteSettings handles GET /profile/settings
func GetSiteSettings(c *fiber.Ctx) error {
	settings, err := services.Profiles().GetSiteSettings()
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
	isAdmin, err := services.Profiles().IsUserAdmin(userID)
	if err != nil {
		return response.Error(c, err)
	}
	if !isAdmin {
		return response.Error(c, core.ForbiddenError("Only admins can update site settings"))
	}

	var req services.SiteSettingsUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}

	settings, err := services.Profiles().UpdateSiteSettings(req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, settings)
}
