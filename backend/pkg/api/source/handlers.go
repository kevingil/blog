package source

import (
	"sync"

	"backend/pkg/api/response"
	"backend/pkg/api/validation"
	"backend/pkg/core"
	coreSource "backend/pkg/core/source"
	"backend/pkg/database"
	"backend/pkg/database/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

var (
	serviceInstance *coreSource.Service
	serviceOnce     sync.Once
)

// getService returns the source service instance (lazily initialized)
func getService() *coreSource.Service {
	serviceOnce.Do(func() {
		db := database.DB()
		sourceRepo := repository.NewSourceRepository(db)
		articleRepo := repository.NewArticleRepository(db)
		serviceInstance = coreSource.NewService(sourceRepo, articleRepo)
	})
	return serviceInstance
}

// ListAllSources handles GET /dashboard/sources
// @Summary List all sources
// @Description Get a paginated list of all sources
// @Tags sources
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /dashboard/sources [get]
func ListAllSources(c *fiber.Ctx) error {
	svc := getService()
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)

	result, err := svc.List(c.Context(), page, limit)
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
// @Summary Create source
// @Description Create a new article source
// @Tags sources
// @Accept json
// @Produce json
// @Param request body dto.CreateSourceRequest true "Source details"
// @Success 201 {object} response.SuccessResponse{data=dto.SourceResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 401 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /sources [post]
func CreateSource(c *fiber.Ctx) error {
	svc := getService()
	var req coreSource.CreateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}
	if err := validation.ValidateStruct(req); err != nil {
		return response.Error(c, err)
	}

	source, err := svc.Create(c.Context(), req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, source)
}

// ScrapeAndCreateSource handles POST /sources/scrape
// @Summary Scrape and create source
// @Description Scrape a URL and create a source from it
// @Tags sources
// @Accept json
// @Produce json
// @Param request body object{article_id=string,url=string} true "Scrape request"
// @Success 201 {object} response.SuccessResponse{data=dto.SourceResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 401 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /sources/scrape [post]
func ScrapeAndCreateSource(c *fiber.Ctx) error {
	svc := getService()
	var req struct {
		ArticleID uuid.UUID `json:"article_id" validate:"required"`
		URL       string    `json:"url" validate:"required,url"`
	}

	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}
	if err := validation.ValidateStruct(req); err != nil {
		return response.Error(c, err)
	}

	source, err := svc.ScrapeAndCreate(c.Context(), req.ArticleID, req.URL)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, source)
}

// GetArticleSources handles GET /sources/article/:articleId
// @Summary Get article sources
// @Description Get all sources for an article
// @Tags sources
// @Accept json
// @Produce json
// @Param articleId path string true "Article ID"
// @Success 200 {object} response.SuccessResponse{data=object{sources=[]dto.SourceResponse}}
// @Failure 400 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /sources/article/{articleId} [get]
func GetArticleSources(c *fiber.Ctx) error {
	svc := getService()
	articleIDStr := c.Params("articleId")
	articleID, err := uuid.Parse(articleIDStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid article ID"))
	}

	sources, err := svc.GetByArticleID(c.Context(), articleID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{"sources": sources})
}

// GetSource handles GET /sources/:sourceId
// @Summary Get source
// @Description Get a source by ID
// @Tags sources
// @Accept json
// @Produce json
// @Param sourceId path string true "Source ID"
// @Success 200 {object} response.SuccessResponse{data=dto.SourceResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /sources/{sourceId} [get]
func GetSource(c *fiber.Ctx) error {
	svc := getService()
	sourceIDStr := c.Params("sourceId")
	sourceID, err := uuid.Parse(sourceIDStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid source ID"))
	}

	source, err := svc.GetByID(c.Context(), sourceID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, source)
}

// UpdateSource handles PUT /sources/:sourceId
// @Summary Update source
// @Description Update an existing source
// @Tags sources
// @Accept json
// @Produce json
// @Param sourceId path string true "Source ID"
// @Param request body dto.UpdateSourceRequest true "Source update details"
// @Success 200 {object} response.SuccessResponse{data=dto.SourceResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /sources/{sourceId} [put]
func UpdateSource(c *fiber.Ctx) error {
	svc := getService()
	sourceIDStr := c.Params("sourceId")
	sourceID, err := uuid.Parse(sourceIDStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid source ID"))
	}

	var req coreSource.UpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}
	if err := validation.ValidateStruct(req); err != nil {
		return response.Error(c, err)
	}

	source, err := svc.Update(c.Context(), sourceID, req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, source)
}

// DeleteSource handles DELETE /sources/:sourceId
// @Summary Delete source
// @Description Delete a source by ID
// @Tags sources
// @Accept json
// @Produce json
// @Param sourceId path string true "Source ID"
// @Success 200 {object} response.SuccessResponse{data=object{success=boolean}}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /sources/{sourceId} [delete]
func DeleteSource(c *fiber.Ctx) error {
	svc := getService()
	sourceIDStr := c.Params("sourceId")
	sourceID, err := uuid.Parse(sourceIDStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid source ID"))
	}

	if err := svc.Delete(c.Context(), sourceID); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"success": true})
}

// SearchSimilarSources handles GET /sources/article/:articleId/search
// @Summary Search similar sources
// @Description Search for similar sources within an article using semantic search
// @Tags sources
// @Accept json
// @Produce json
// @Param articleId path string true "Article ID"
// @Param q query string true "Search query"
// @Param limit query int false "Max results" default(5)
// @Success 200 {object} response.SuccessResponse{data=object{sources=[]dto.SourceResponse,query=string}}
// @Failure 400 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /sources/article/{articleId}/search [get]
func SearchSimilarSources(c *fiber.Ctx) error {
	svc := getService()
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

	sources, err := svc.SearchSimilar(c.Context(), articleID, query, limit)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{
		"sources": sources,
		"query":   query,
	})
}
