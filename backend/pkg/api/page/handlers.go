package page

import (
	"backend/pkg/api/response"
	"backend/pkg/api/validation"
	"backend/pkg/core"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GetPageBySlug handles GET /pages/:slug
func GetPageBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return response.Error(c, core.InvalidInputError("Page slug is required"))
	}

	page, err := Pages().GetPageBySlug(slug)
	if err != nil {
		return response.Error(c, err)
	}
	if page == nil {
		return response.Error(c, core.NotFoundError("Page"))
	}
	return response.Success(c, page)
}

// ListPages handles GET /dashboard/pages
func ListPages(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	perPage := c.QueryInt("perPage", 20)

	var isPublished *bool
	isPublishedStr := c.Query("isPublished")
	if isPublishedStr != "" {
		val := c.QueryBool("isPublished", true)
		isPublished = &val
	}

	pages, err := Pages().ListPagesWithPagination(page, perPage, isPublished)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, pages)
}

// GetPageByID handles GET /dashboard/pages/:id
func GetPageByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid page ID"))
	}

	page, err := Pages().GetPageByID(id)
	if err != nil {
		return response.Error(c, err)
	}
	if page == nil {
		return response.Error(c, core.NotFoundError("Page"))
	}
	return response.Success(c, page)
}

// CreatePage handles POST /dashboard/pages
func CreatePage(c *fiber.Ctx) error {
	var req PageCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}
	if err := validation.ValidateStruct(req); err != nil {
		return response.Error(c, err)
	}

	page, err := Pages().CreatePage(req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, page)
}

// UpdatePage handles PUT /dashboard/pages/:id
func UpdatePage(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid page ID"))
	}

	var req PageUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}

	page, err := Pages().UpdatePage(id, req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, page)
}

// DeletePage handles DELETE /dashboard/pages/:id
func DeletePage(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid page ID"))
	}

	if err := Pages().DeletePage(id); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"success": true})
}
