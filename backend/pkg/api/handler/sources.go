package handler

import (
	"backend/pkg/api/response"
	"backend/pkg/api/services"
	"backend/pkg/core"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ListAllSourcesHandler retrieves all sources with pagination and article metadata
func ListAllSourcesHandler(sourcesService *services.ArticleSourceService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		page := c.QueryInt("page", 1)
		limit := c.QueryInt("limit", 20)

		sources, totalPages, err := sourcesService.GetAllSources(page, limit)
		if err != nil {
			return response.Error(c, err)
		}

		return response.Success(c, fiber.Map{
			"sources":     sources,
			"total_pages": totalPages,
			"page":        page,
		})
	}
}

// CreateSourceHandler creates a new article source
func CreateSourceHandler(sourcesService *services.ArticleSourceService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req services.CreateSourceRequest
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, core.InvalidInputError("Invalid request body"))
		}

		// Validate required fields
		if req.ArticleID == uuid.Nil {
			return response.Error(c, core.InvalidInputError("Article ID is required"))
		}

		if req.Content == "" {
			return response.Error(c, core.InvalidInputError("Content is required"))
		}

		source, err := sourcesService.CreateSource(c.Context(), req)
		if err != nil {
			return response.Error(c, err)
		}

		return response.Created(c, source)
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
			return response.Error(c, core.InvalidInputError("Invalid request body"))
		}

		// Validate required fields
		if req.ArticleID == uuid.Nil {
			return response.Error(c, core.InvalidInputError("Article ID is required"))
		}

		if req.URL == "" {
			return response.Error(c, core.InvalidInputError("URL is required"))
		}

		source, err := sourcesService.ScrapeAndCreateSource(c.Context(), req.ArticleID, req.URL)
		if err != nil {
			return response.Error(c, err)
		}

		return response.Created(c, source)
	}
}

// GetArticleSourcesHandler retrieves all sources for an article
func GetArticleSourcesHandler(sourcesService *services.ArticleSourceService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		articleIDStr := c.Params("articleId")
		articleID, err := uuid.Parse(articleIDStr)
		if err != nil {
			return response.Error(c, core.InvalidInputError("Invalid article ID"))
		}

		sources, err := sourcesService.GetSourcesByArticleID(articleID)
		if err != nil {
			return response.Error(c, err)
		}

		return response.Success(c, fiber.Map{
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
			return response.Error(c, core.InvalidInputError("Invalid source ID"))
		}

		source, err := sourcesService.GetSource(sourceID)
		if err != nil {
			return response.Error(c, err)
		}

		return response.Success(c, source)
	}
}

// UpdateSourceHandler updates an existing source
func UpdateSourceHandler(sourcesService *services.ArticleSourceService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sourceIDStr := c.Params("sourceId")
		sourceID, err := uuid.Parse(sourceIDStr)
		if err != nil {
			return response.Error(c, core.InvalidInputError("Invalid source ID"))
		}

		var req services.UpdateSourceRequest
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, core.InvalidInputError("Invalid request body"))
		}

		source, err := sourcesService.UpdateSource(c.Context(), sourceID, req)
		if err != nil {
			return response.Error(c, err)
		}

		return response.Success(c, source)
	}
}

// DeleteSourceHandler deletes a source
func DeleteSourceHandler(sourcesService *services.ArticleSourceService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sourceIDStr := c.Params("sourceId")
		sourceID, err := uuid.Parse(sourceIDStr)
		if err != nil {
			return response.Error(c, core.InvalidInputError("Invalid source ID"))
		}

		err = sourcesService.DeleteSource(sourceID)
		if err != nil {
			return response.Error(c, err)
		}

		return response.Success(c, fiber.Map{"success": true})
	}
}

// SearchSimilarSourcesHandler searches for similar sources using vector similarity
func SearchSimilarSourcesHandler(sourcesService *services.ArticleSourceService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		articleIDStr := c.Params("articleId")
		articleID, err := uuid.Parse(articleIDStr)
		if err != nil {
			return response.Error(c, core.InvalidInputError("Invalid article ID"))
		}

		query := c.Query("q", "")
		if query == "" {
			return response.Error(c, core.InvalidInputError("Query parameter 'q' is required"))
		}

		limit := c.QueryInt("limit", 5)
		if limit > 20 {
			limit = 20 // Cap the limit
		}

		sources, err := sourcesService.SearchSimilarSources(c.Context(), articleID, query, limit)
		if err != nil {
			return response.Error(c, err)
		}

		return response.Success(c, fiber.Map{
			"sources": sources,
			"query":   query,
		})
	}
}
