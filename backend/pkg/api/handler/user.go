package handler

import (
	"backend/pkg/core"
	"backend/pkg/api/response"
	"backend/pkg/api/services"
	"backend/pkg/api/validation"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func LoginHandler(authService *services.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req services.LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, core.InvalidInputError("Invalid request body"))
		}
		if err := validation.ValidateStruct(req); err != nil {
			return response.Error(c, err)
		}
		resp, err := authService.Login(req)
		if err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, resp)
	}
}

func RegisterHandler(authService *services.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req services.RegisterRequest
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, core.InvalidInputError("Invalid request body"))
		}
		if err := validation.ValidateStruct(req); err != nil {
			return response.Error(c, err)
		}
		if err := authService.Register(req); err != nil {
			return response.Error(c, err)
		}
		return response.Created(c, fiber.Map{
			"message": "User registered successfully",
		})
	}
}

func LogoutHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return response.Success(c, fiber.Map{
			"message": "Logged out successfully",
		})
	}
}

func UpdateAccountHandler(authService *services.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		var req services.UpdateAccountRequest
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, core.InvalidInputError("Invalid request body"))
		}
		if err := validation.ValidateStruct(req); err != nil {
			return response.Error(c, err)
		}
		if err := authService.UpdateAccount(userID, req); err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, fiber.Map{"message": "Account updated successfully"})
	}
}

func UpdatePasswordHandler(authService *services.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		var req services.UpdatePasswordRequest
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, core.InvalidInputError("Invalid request body"))
		}
		if err := validation.ValidateStruct(req); err != nil {
			return response.Error(c, err)
		}
		if err := authService.UpdatePassword(userID, req); err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, fiber.Map{"message": "Password updated successfully"})
	}
}

func DeleteAccountHandler(authService *services.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		var req struct {
			Password string `json:"password"`
		}
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, core.InvalidInputError("Invalid request body"))
		}
		if err := authService.DeleteAccount(userID, req.Password); err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, fiber.Map{"message": "Account deleted successfully"})
	}
}
