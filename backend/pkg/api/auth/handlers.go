package auth

import (
	"backend/pkg/api/middleware"
	"backend/pkg/api/response"
	"backend/pkg/api/validation"
	"backend/pkg/core"
	coreAuth "backend/pkg/core/auth"

	"github.com/gofiber/fiber/v2"
)

// Login handles POST /auth/login
// @Summary User login
// @Description Authenticate user and return JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "Login credentials"
// @Success 200 {object} response.SuccessResponse{data=dto.LoginResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 401 {object} response.SuccessResponse
// @Router /auth/login [post]
func Login(c *fiber.Ctx) error {
	var req coreAuth.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}
	if err := validation.ValidateStruct(req); err != nil {
		return response.Error(c, err)
	}

	resp, err := coreAuth.Login(c.Context(), req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, resp)
}

// RegisterHandler handles POST /auth/register
// @Summary User registration
// @Description Register a new user account
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "Registration details"
// @Success 201 {object} response.SuccessResponse{data=object{message=string}}
// @Failure 400 {object} response.SuccessResponse
// @Failure 409 {object} response.SuccessResponse
// @Router /auth/register [post]
func RegisterHandler(c *fiber.Ctx) error {
	var req coreAuth.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}
	if err := validation.ValidateStruct(req); err != nil {
		return response.Error(c, err)
	}

	if err := coreAuth.Register(c.Context(), req); err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, fiber.Map{
		"message": "User registered successfully",
	})
}

// Logout handles POST /auth/logout
// @Summary User logout
// @Description Logout current user session
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} response.SuccessResponse{data=object{message=string}}
// @Security BearerAuth
// @Router /auth/logout [post]
func Logout(c *fiber.Ctx) error {
	return response.Success(c, fiber.Map{
		"message": "Logged out successfully",
	})
}

// UpdateAccount handles PUT /auth/account
// @Summary Update account
// @Description Update current user's account details
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.UpdateAccountRequest true "Account update details"
// @Success 200 {object} response.SuccessResponse{data=object{message=string}}
// @Failure 400 {object} response.SuccessResponse
// @Failure 401 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /auth/account [put]
func UpdateAccount(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return response.Error(c, core.UnauthorizedError("Not authenticated"))
	}

	var req coreAuth.UpdateAccountRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}
	if err := validation.ValidateStruct(req); err != nil {
		return response.Error(c, err)
	}

	if err := coreAuth.UpdateAccount(c.Context(), userID, req); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"message": "Account updated successfully"})
}

// UpdatePassword handles PUT /auth/password
// @Summary Update password
// @Description Update current user's password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.UpdatePasswordRequest true "Password update details"
// @Success 200 {object} response.SuccessResponse{data=object{message=string}}
// @Failure 400 {object} response.SuccessResponse
// @Failure 401 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /auth/password [put]
func UpdatePassword(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return response.Error(c, core.UnauthorizedError("Not authenticated"))
	}

	var req coreAuth.UpdatePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}
	if err := validation.ValidateStruct(req); err != nil {
		return response.Error(c, err)
	}

	if err := coreAuth.UpdatePassword(c.Context(), userID, req); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"message": "Password updated successfully"})
}

// DeleteAccount handles DELETE /auth/account
// @Summary Delete account
// @Description Permanently delete current user's account
// @Tags auth
// @Accept json
// @Produce json
// @Param request body object{password=string} true "Account deletion confirmation"
// @Success 200 {object} response.SuccessResponse{data=object{message=string}}
// @Failure 400 {object} response.SuccessResponse
// @Failure 401 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /auth/account [delete]
func DeleteAccount(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return response.Error(c, core.UnauthorizedError("Not authenticated"))
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}

	if err := coreAuth.DeleteAccount(c.Context(), userID, req.Password); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"message": "Account deleted successfully"})
}
