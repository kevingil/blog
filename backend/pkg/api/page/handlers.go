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
