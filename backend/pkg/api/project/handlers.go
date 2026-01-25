package project

import (
	"backend/pkg/api/middleware"
	"backend/pkg/api/response"
	"backend/pkg/api/validation"
	"backend/pkg/core"
	"backend/pkg/core/project"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ListProjects handles GET /projects
func ListProjects(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	perPage := c.QueryInt("perPage", 20)

	result, err := project.List(c.Context(), page, perPage)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{
		"projects":     result.Projects,
		"total":        result.Total,
		"current_page": result.Page,
		"per_page":     result.PerPage,
	})
}

// GetProject handles GET /projects/:id
func GetProject(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid project ID"))
	}

	detail, err := project.GetDetail(c.Context(), id)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, detail)
}

// CreateProject handles POST /projects
func CreateProject(c *fiber.Ctx) error {
	var req project.CreateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}

	if err := validation.ValidateStruct(req); err != nil {
		return response.Error(c, err)
	}

	result, err := project.Create(c.Context(), req)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, result)
}

// UpdateProject handles PUT /projects/:id
func UpdateProject(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid project ID"))
	}

	var req project.UpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}

	result, err := project.Update(c.Context(), id, req)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, result)
}

// DeleteProject handles DELETE /projects/:id
func DeleteProject(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid project ID"))
	}

	// Verify user is authenticated (already done by middleware, but we can get userID if needed)
	_, err = middleware.GetUserID(c)
	if err != nil {
		return response.Error(c, core.UnauthorizedError("Not authenticated"))
	}

	if err := project.Delete(c.Context(), id); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{"success": true})
}
