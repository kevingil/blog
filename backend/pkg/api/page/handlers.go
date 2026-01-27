package page

import (
	"backend/pkg/api/response"
	"backend/pkg/api/validation"
	"backend/pkg/core"
	corePage "backend/pkg/core/page"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GetPageBySlug handles GET /pages/:slug
// @Summary Get page by slug
// @Description Get a public page by its slug
// @Tags pages
// @Accept json
// @Produce json
// @Param slug path string true "Page slug"
// @Success 200 {object} response.SuccessResponse{data=dto.PageResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Router /pages/{slug} [get]
func GetPageBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return response.Error(c, core.InvalidInputError("Page slug is required"))
	}

	page, err := corePage.GetBySlug(c.Context(), slug)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, page)
}

// ListPages handles GET /dashboard/pages
// @Summary List pages
// @Description Get a paginated list of pages (dashboard)
// @Tags pages
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param perPage query int false "Items per page" default(20)
// @Param isPublished query bool false "Filter by published status"
// @Success 200 {object} response.SuccessResponse{data=dto.PageListResponse}
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /dashboard/pages [get]
func ListPages(c *fiber.Ctx) error {
	pageNum := c.QueryInt("page", 1)
	perPage := c.QueryInt("perPage", 20)

	var isPublished *bool
	isPublishedStr := c.Query("isPublished")
	if isPublishedStr != "" {
		val := c.QueryBool("isPublished", true)
		isPublished = &val
	}

	result, err := corePage.List(c.Context(), pageNum, perPage, isPublished)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, result)
}

// GetPageByID handles GET /dashboard/pages/:id
// @Summary Get page by ID
// @Description Get a page by its ID (dashboard)
// @Tags pages
// @Accept json
// @Produce json
// @Param id path string true "Page ID"
// @Success 200 {object} response.SuccessResponse{data=dto.PageResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /dashboard/pages/{id} [get]
func GetPageByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid page ID"))
	}

	page, err := corePage.GetByID(c.Context(), id)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, page)
}

// CreatePage handles POST /dashboard/pages
// @Summary Create page
// @Description Create a new page
// @Tags pages
// @Accept json
// @Produce json
// @Param request body dto.CreatePageRequest true "Page details"
// @Success 201 {object} response.SuccessResponse{data=dto.PageResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 401 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /dashboard/pages [post]
func CreatePage(c *fiber.Ctx) error {
	var req corePage.CreateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}
	if err := validation.ValidateStruct(req); err != nil {
		return response.Error(c, err)
	}

	page, err := corePage.Create(c.Context(), req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, page)
}

// UpdatePage handles PUT /dashboard/pages/:id
// @Summary Update page
// @Description Update an existing page
// @Tags pages
// @Accept json
// @Produce json
// @Param id path string true "Page ID"
// @Param request body dto.UpdatePageRequest true "Page update details"
// @Success 200 {object} response.SuccessResponse{data=dto.PageResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /dashboard/pages/{id} [put]
func UpdatePage(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid page ID"))
	}

	var req corePage.UpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}

	page, err := corePage.Update(c.Context(), id, req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, page)
}

// DeletePage handles DELETE /dashboard/pages/:id
// @Summary Delete page
// @Description Delete a page by ID
// @Tags pages
// @Accept json
// @Produce json
// @Param id path string true "Page ID"
// @Success 200 {object} response.SuccessResponse{data=object{success=boolean}}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /dashboard/pages/{id} [delete]
func DeletePage(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid page ID"))
	}

	if err := corePage.Delete(c.Context(), id); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"success": true})
}
