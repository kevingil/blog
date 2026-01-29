package datasource

import (
	"backend/pkg/api/middleware"
	"backend/pkg/api/response"
	"backend/pkg/api/validation"
	"backend/pkg/core"
	coreDS "backend/pkg/core/datasource"
	"backend/pkg/types"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ListDataSources handles GET /data-sources
// @Summary List data sources
// @Description Get a list of all data sources for the authenticated user's organization
// @Tags data-sources
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} response.SuccessResponse{data=[]types.DataSourceResponse}
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /data-sources [get]
func ListDataSources(c *fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)

	if orgID != nil {
		sources, err := coreDS.List(c.Context(), *orgID)
		if err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, sources)
	}

	// If no org, return all with pagination
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)

	sources, total, err := coreDS.ListAll(c.Context(), page, limit)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{
		"data_sources": sources,
		"total":        total,
		"page":         page,
		"limit":        limit,
	})
}

// GetDataSource handles GET /data-sources/:id
// @Summary Get data source
// @Description Get a data source by ID
// @Tags data-sources
// @Accept json
// @Produce json
// @Param id path string true "Data Source ID"
// @Success 200 {object} response.SuccessResponse{data=types.DataSourceResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /data-sources/{id} [get]
func GetDataSource(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid data source ID"))
	}

	ds, err := coreDS.GetByID(c.Context(), id)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, ds)
}

// CreateDataSource handles POST /data-sources
// @Summary Create data source
// @Description Create a new data source (preferred website to crawl)
// @Tags data-sources
// @Accept json
// @Produce json
// @Param request body types.DataSourceCreateRequest true "Data source details"
// @Success 201 {object} response.SuccessResponse{data=types.DataSourceResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 401 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /data-sources [post]
func CreateDataSource(c *fiber.Ctx) error {
	var req types.DataSourceCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}
	if err := validation.ValidateStruct(req); err != nil {
		return response.Error(c, err)
	}

	orgID := middleware.GetOrgID(c)

	ds, err := coreDS.Create(c.Context(), orgID, req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, ds)
}

// UpdateDataSource handles PUT /data-sources/:id
// @Summary Update data source
// @Description Update an existing data source
// @Tags data-sources
// @Accept json
// @Produce json
// @Param id path string true "Data Source ID"
// @Param request body types.DataSourceUpdateRequest true "Data source update details"
// @Success 200 {object} response.SuccessResponse{data=types.DataSourceResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /data-sources/{id} [put]
func UpdateDataSource(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid data source ID"))
	}

	var req types.DataSourceUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}

	ds, err := coreDS.Update(c.Context(), id, req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, ds)
}

// DeleteDataSource handles DELETE /data-sources/:id
// @Summary Delete data source
// @Description Delete a data source by ID
// @Tags data-sources
// @Accept json
// @Produce json
// @Param id path string true "Data Source ID"
// @Success 200 {object} response.SuccessResponse{data=object{success=boolean}}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /data-sources/{id} [delete]
func DeleteDataSource(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid data source ID"))
	}

	if err := coreDS.Delete(c.Context(), id); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"success": true})
}

// TriggerCrawl handles POST /data-sources/:id/crawl
// @Summary Trigger crawl
// @Description Trigger a manual crawl for a data source
// @Tags data-sources
// @Accept json
// @Produce json
// @Param id path string true "Data Source ID"
// @Success 200 {object} response.SuccessResponse{data=object{success=boolean,message=string}}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /data-sources/{id}/crawl [post]
func TriggerCrawl(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid data source ID"))
	}

	if err := coreDS.TriggerCrawl(c.Context(), id); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{
		"success": true,
		"message": "Crawl triggered successfully",
	})
}

// GetDataSourceContent handles GET /data-sources/:id/content
// @Summary Get data source content
// @Description Get crawled content for a data source
// @Tags data-sources
// @Accept json
// @Produce json
// @Param id path string true "Data Source ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} response.SuccessResponse{data=object{contents=[]types.CrawledContentResponse,total=int64}}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /data-sources/{id}/content [get]
func GetDataSourceContent(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid data source ID"))
	}

	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)

	contents, total, err := coreDS.GetContent(c.Context(), id, page, limit)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{
		"contents": contents,
		"total":    total,
		"page":     page,
		"limit":    limit,
	})
}
