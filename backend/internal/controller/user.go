package controller

import (
	"blog-agent-go/backend/internal/services"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func LoginHandler(authService *services.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req services.LoginRequest
		if err := c.BodyParser(&req); err != nil {
			fmt.Println("Error parsing request body:", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}
		resp, err := authService.Login(req)
		if err != nil {
			fmt.Println("Error logging in:", err)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.JSON(resp)
	}
}

func RegisterHandler(authService *services.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req services.RegisterRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}
		if err := authService.Register(req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"message": "User registered successfully",
		})
	}
}

func LogoutHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Logged out successfully",
		})
	}
}

func UpdateAccountHandler(authService *services.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		var req services.UpdateAccountRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		if err := authService.UpdateAccount(userID, req); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "Account updated successfully"})
	}
}

func UpdatePasswordHandler(authService *services.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		var req services.UpdatePasswordRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		if err := authService.UpdatePassword(userID, req); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "Password updated successfully"})
	}
}

func DeleteAccountHandler(authService *services.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)
		var req struct {
			Password string `json:"password"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		if err := authService.DeleteAccount(userID, req.Password); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "Account deleted successfully"})
	}
}

func AuthMiddleware(authService *services.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Get("Authorization")
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Not authenticated"})
		}
		if len(token) < 7 || token[:7] != "Bearer " {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token format"})
		}
		token = token[7:]
		validToken, err := authService.ValidateToken(token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
		}
		claims := validToken.Claims.(jwt.MapClaims)
		c.Locals("userID", uuid.MustParse(claims["sub"].(string)))
		return c.Next()
	}
}
