package source

import (
	"backend/pkg/api/response"
	"backend/pkg/core"
	coreSource "backend/pkg/core/source"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ListAllSources handles GET /dashboard/sources
func ListAllSources(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)

	result, err := coreSource.List(c.Context(), page, limit)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{
		"sources":     result.Sources,
		"total_pages": result.TotalPages,
		"page":        result.Page,
	})
}

// CreateSource handles POST /sources
func CreateSource(c *fiber.Ctx) error {
	var req coreSource.CreateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}

	if req.ArticleID == uuid.Nil {
		return response.Error(c, core.InvalidInputError("Article ID is required"))
	}
	if req.Content == "" {
		return response.Error(c, core.InvalidInputError("Content is required"))
	}

	source, err := coreSource.Create(c.Context(), req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, source)
}

// ScrapeAndCreateSource handles POST /sources/scrape
func ScrapeAndCreateSource(c *fiber.Ctx) error {
	var req struct {
		ArticleID uuid.UUID `json:"article_id"`
		URL       string    `json:"url"`
	}

	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}

	if req.ArticleID == uuid.Nil {
		return response.Error(c, core.InvalidInputError("Article ID is required"))
	}
	if req.URL == "" {
		return response.Error(c, core.InvalidInputError("URL is required"))
	}

	source, err := coreSource.ScrapeAndCreate(c.Context(), req.ArticleID, req.URL)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, source)
}

// GetArticleSources handles GET /sources/article/:articleId
func GetArticleSources(c *fiber.Ctx) error {
	articleIDStr := c.Params("articleId")
	articleID, err := uuid.Parse(articleIDStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid article ID"))
	}

	sources, err := coreSource.GetByArticleID(c.Context(), articleID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{"sources": sources})
}

// GetSource handles GET /sources/:sourceId
func GetSource(c *fiber.Ctx) error {
	sourceIDStr := c.Params("sourceId")
	sourceID, err := uuid.Parse(sourceIDStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid source ID"))
	}

	source, err := coreSource.GetByID(c.Context(), sourceID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, source)
}

// UpdateSource handles PUT /sources/:sourceId
func UpdateSource(c *fiber.Ctx) error {
	sourceIDStr := c.Params("sourceId")
	sourceID, err := uuid.Parse(sourceIDStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid source ID"))
	}

	var req coreSource.UpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}

	source, err := coreSource.Update(c.Context(), sourceID, req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, source)
}

// DeleteSource handles DELETE /sources/:sourceId
func DeleteSource(c *fiber.Ctx) error {
	sourceIDStr := c.Params("sourceId")
	sourceID, err := uuid.Parse(sourceIDStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid source ID"))
	}

	if err := coreSource.Delete(c.Context(), sourceID); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"success": true})
}

// SearchSimilarSources handles GET /sources/article/:articleId/search
func SearchSimilarSources(c *fiber.Ctx) error {
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
		limit = 20
	}

	sources, err := coreSource.SearchSimilar(c.Context(), articleID, query, limit)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{
		"sources": sources,
		"query":   query,
	})
}
