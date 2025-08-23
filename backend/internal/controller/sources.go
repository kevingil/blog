package controller

import (
	"blog-agent-go/backend/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreateSourceHandler creates a new article source
func CreateSourceHandler(sourcesService *services.ArticleSourceService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req services.CreateSourceRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		// Validate required fields
		if req.ArticleID == uuid.Nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Article ID is required",
			})
		}

		if req.Content == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Content is required",
			})
		}

		source, err := sourcesService.CreateSource(c.Context(), req)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.Status(fiber.StatusCreated).JSON(source)
	}
}

// ScrapeAndCreateSourceHandler scrapes a URL and creates a source
func ScrapeAndCreateSourceHandler(sourcesService *services.ArticleSourceService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			ArticleID uuid.UUID `json:"article_id"`
			URL       string    `json:"url"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		// Validate required fields
		if req.ArticleID == uuid.Nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Article ID is required",
			})
		}

		if req.URL == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "URL is required",
			})
		}

		source, err := sourcesService.ScrapeAndCreateSource(c.Context(), req.ArticleID, req.URL)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.Status(fiber.StatusCreated).JSON(source)
	}
}

// GetArticleSourcesHandler retrieves all sources for an article
func GetArticleSourcesHandler(sourcesService *services.ArticleSourceService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		articleIDStr := c.Params("articleId")
		articleID, err := uuid.Parse(articleIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid article ID",
			})
		}

		sources, err := sourcesService.GetSourcesByArticleID(articleID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"sources": sources,
		})
	}
}

// GetSourceHandler retrieves a specific source by ID
func GetSourceHandler(sourcesService *services.ArticleSourceService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sourceIDStr := c.Params("sourceId")
		sourceID, err := uuid.Parse(sourceIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid source ID",
			})
		}

		source, err := sourcesService.GetSource(sourceID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "Source not found",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(source)
	}
}

// UpdateSourceHandler updates an existing source
func UpdateSourceHandler(sourcesService *services.ArticleSourceService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sourceIDStr := c.Params("sourceId")
		sourceID, err := uuid.Parse(sourceIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid source ID",
			})
		}

		var req services.UpdateSourceRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		source, err := sourcesService.UpdateSource(c.Context(), sourceID, req)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "Source not found",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(source)
	}
}

// DeleteSourceHandler deletes a source
func DeleteSourceHandler(sourcesService *services.ArticleSourceService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sourceIDStr := c.Params("sourceId")
		sourceID, err := uuid.Parse(sourceIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid source ID",
			})
		}

		err = sourcesService.DeleteSource(sourceID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "Source not found",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.Status(fiber.StatusNoContent).Send(nil)
	}
}

// SearchSimilarSourcesHandler searches for similar sources using vector similarity
func SearchSimilarSourcesHandler(sourcesService *services.ArticleSourceService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		articleIDStr := c.Params("articleId")
		articleID, err := uuid.Parse(articleIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid article ID",
			})
		}

		query := c.Query("q", "")
		if query == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Query parameter 'q' is required",
			})
		}

		limit := c.QueryInt("limit", 5)
		if limit > 20 {
			limit = 20 // Cap the limit
		}

		sources, err := sourcesService.SearchSimilarSources(c.Context(), articleID, query, limit)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"sources": sources,
			"query":   query,
		})
	}
}
