package handler

import (
	"backend/pkg/api/middleware"
	"backend/pkg/api/response"
	"backend/pkg/api/services"
	"backend/pkg/core"

	"github.com/gofiber/fiber/v2"
)

// GetPublicProfileHandler returns the public profile based on site settings
func GetPublicProfileHandler(profileService *services.ProfileService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		profile, err := profileService.GetPublicProfile()
		if err != nil {
			return response.Error(c, err)
		}
		if profile == nil {
			return response.Error(c, core.NotFoundError("Public profile"))
		}
		return response.Success(c, profile)
	}
}

// GetMyProfileHandler returns the authenticated user's profile
func GetMyProfileHandler(profileService *services.ProfileService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, err := middleware.GetUserID(c)
		if err != nil {
			return response.Error(c, core.UnauthorizedError("Not authenticated"))
		}

		profile, err := profileService.GetUserProfile(userID)
		if err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, profile)
	}
}

// UpdateProfileHandler updates the authenticated user's profile
func UpdateProfileHandler(profileService *services.ProfileService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, err := middleware.GetUserID(c)
		if err != nil {
			return response.Error(c, core.UnauthorizedError("Not authenticated"))
		}

		var req services.ProfileUpdateRequest
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, core.InvalidInputError("Invalid request body"))
		}

		profile, err := profileService.UpdateUserProfile(userID, req)
		if err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, profile)
	}
}

// GetSiteSettingsHandler returns the current site settings
func GetSiteSettingsHandler(profileService *services.ProfileService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		settings, err := profileService.GetSiteSettings()
		if err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, settings)
	}
}

// UpdateSiteSettingsHandler updates the site settings (admin only)
func UpdateSiteSettingsHandler(profileService *services.ProfileService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, err := middleware.GetUserID(c)
		if err != nil {
			return response.Error(c, core.UnauthorizedError("Not authenticated"))
		}

		// Check if user is admin
		isAdmin, err := profileService.IsUserAdmin(userID)
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

		settings, err := profileService.UpdateSiteSettings(req)
		if err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, settings)
	}
}
