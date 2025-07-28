package controller

import (
	"blog-agent-go/backend/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func GenerateArticleImageHandler(imageService *services.ImageGenerationService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			Prompt         string    `json:"prompt"`
			ArticleID      uuid.UUID `json:"article_id"`
			GeneratePrompt bool      `json:"generate_prompt"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		imageGen, err := imageService.GenerateArticleImage(c.Context(), req.Prompt, req.ArticleID, req.GeneratePrompt)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(imageGen)
	}
}

func GetImageGenerationHandler(imageService *services.ImageGenerationService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestID := c.Params("requestId")
		if requestID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request ID"})
		}
		imageGen, err := imageService.GetImageGeneration(c.Context(), requestID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(imageGen)
	}
}

func GetImageGenerationStatusHandler(imageService *services.ImageGenerationService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestID := c.Params("requestId")
		if requestID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request ID"})
		}
		status, err := imageService.GetImageGenerationStatus(c.Context(), requestID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(status)
	}
}
