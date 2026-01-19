package controller

import (
	"blog-agent-go/backend/internal/errors"
	"blog-agent-go/backend/internal/middleware"
	"blog-agent-go/backend/internal/response"
	"blog-agent-go/backend/internal/services"

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
			return response.Error(c, errors.NewNotFoundError("Public profile"))
		}
		return response.Success(c, profile)
	}
}

// GetMyProfileHandler returns the authenticated user's profile
func GetMyProfileHandler(profileService *services.ProfileService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, err := middleware.GetUserID(c)
		if err != nil {
			return response.Error(c, errors.NewUnauthorizedError("Not authenticated"))
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
			return response.Error(c, errors.NewUnauthorizedError("Not authenticated"))
		}

		var req services.ProfileUpdateRequest
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid request body"))
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

// UpdateSiteSettingsHandler updates the site settings
func UpdateSiteSettingsHandler(profileService *services.ProfileService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req services.SiteSettingsUpdateRequest
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid request body"))
		}

		settings, err := profileService.UpdateSiteSettings(req)
		if err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, settings)
	}
}
