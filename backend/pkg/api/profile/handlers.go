package profile

import (
	"backend/pkg/api/middleware"
	"backend/pkg/api/response"
	"backend/pkg/core"
	coreProfile "backend/pkg/core/profile"

	"github.com/gofiber/fiber/v2"
)

// GetPublicProfile handles GET /profile/public
// @Summary Get public profile
// @Description Get the public profile information
// @Tags profile
// @Accept json
// @Produce json
// @Success 200 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Router /profile/public [get]
func GetPublicProfile(c *fiber.Ctx) error {
	profile, err := coreProfile.GetPublicProfile(c.Context())
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, profile)
}

// GetMyProfile handles GET /profile
// @Summary Get my profile
// @Description Get the authenticated user's profile
// @Tags profile
// @Accept json
// @Produce json
// @Success 200 {object} response.SuccessResponse
// @Failure 401 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /profile [get]
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
// @Summary Update profile
// @Description Update the authenticated user's profile
// @Tags profile
// @Accept json
// @Produce json
// @Param request body object true "Profile update details"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.SuccessResponse
// @Failure 401 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /profile [put]
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
// @Summary Get site settings
// @Description Get the site settings
// @Tags profile
// @Accept json
// @Produce json
// @Success 200 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Router /profile/settings [get]
func GetSiteSettings(c *fiber.Ctx) error {
	settings, err := coreProfile.GetSiteSettings(c.Context())
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, settings)
}

// UpdateSiteSettings handles PUT /profile/settings
// @Summary Update site settings
// @Description Update the site settings (admin only)
// @Tags profile
// @Accept json
// @Produce json
// @Param request body object true "Site settings update"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.SuccessResponse
// @Failure 401 {object} response.SuccessResponse
// @Failure 403 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /profile/settings [put]
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
