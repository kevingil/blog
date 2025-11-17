package controller

import (
	"blog-agent-go/backend/internal/errors"
	"blog-agent-go/backend/internal/response"
	"blog-agent-go/backend/internal/services"
	"blog-agent-go/backend/internal/validation"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func ListProjectsHandler(svc *services.ProjectsService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		page := c.QueryInt("page", 1)
		perPage := c.QueryInt("perPage", 20)
		projects, total, err := svc.ListProjects(page, perPage)
		if err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, fiber.Map{
			"projects":     projects,
			"total":        total,
			"current_page": page,
			"per_page":     perPage,
		})
	}
}

func GetProjectHandler(svc *services.ProjectsService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid project ID"))
		}
		project, err := svc.GetProjectDetail(id)
		if err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, project)
	}
}

func CreateProjectHandler(svc *services.ProjectsService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req services.ProjectCreateRequest
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid request body"))
		}
		if err := validation.ValidateStruct(req); err != nil {
			return response.Error(c, err)
		}
		project, err := svc.CreateProject(req)
		if err != nil {
			return response.Error(c, err)
		}
		return response.Created(c, project)
	}
}

func UpdateProjectHandler(svc *services.ProjectsService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid project ID"))
		}
		var req services.ProjectUpdateRequest
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid request body"))
		}
		project, err := svc.UpdateProject(id, req)
		if err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, project)
	}
}

func DeleteProjectHandler(svc *services.ProjectsService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid project ID"))
		}
		if err := svc.DeleteProject(id); err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, fiber.Map{"success": true})
	}
}
