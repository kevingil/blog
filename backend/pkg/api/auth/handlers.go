package auth

import (
	"backend/pkg/api/middleware"
	"backend/pkg/api/response"
	"backend/pkg/api/validation"
	"backend/pkg/core"

	"github.com/gofiber/fiber/v2"
)

// Login handles POST /auth/login
func Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}
	if err := validation.ValidateStruct(req); err != nil {
		return response.Error(c, err)
	}

	resp, err := Auth().Login(req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, resp)
}

// Register_ handles POST /auth/register (named with underscore to avoid conflict with package Register function)
func Register_(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}
	if err := validation.ValidateStruct(req); err != nil {
		return response.Error(c, err)
	}

	if err := Auth().Register(req); err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, fiber.Map{
		"message": "User registered successfully",
	})
}

// Logout handles POST /auth/logout
func Logout(c *fiber.Ctx) error {
	return response.Success(c, fiber.Map{
		"message": "Logged out successfully",
	})
}

// UpdateAccount handles PUT /auth/account
func UpdateAccount(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return response.Error(c, core.UnauthorizedError("Not authenticated"))
	}

	var req UpdateAccountRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}
	if err := validation.ValidateStruct(req); err != nil {
		return response.Error(c, err)
	}

	if err := Auth().UpdateAccount(userID, req); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"message": "Account updated successfully"})
}

// UpdatePassword handles PUT /auth/password
func UpdatePassword(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return response.Error(c, core.UnauthorizedError("Not authenticated"))
	}

	var req UpdatePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}
	if err := validation.ValidateStruct(req); err != nil {
		return response.Error(c, err)
	}

	if err := Auth().UpdatePassword(userID, req); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"message": "Password updated successfully"})
}

// DeleteAccount handles DELETE /auth/account
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

	if err := Auth().DeleteAccount(userID, req.Password); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"message": "Account deleted successfully"})
}
