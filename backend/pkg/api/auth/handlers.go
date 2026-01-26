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
