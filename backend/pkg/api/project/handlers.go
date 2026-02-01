package project

import (
	"backend/pkg/api/middleware"
	"backend/pkg/api/response"
	"backend/pkg/api/validation"
	"backend/pkg/core"
	"backend/pkg/core/project"
	"backend/pkg/database"
	"backend/pkg/database/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// getService creates a project service with repository injection
func getService() *project.Service {
	db := database.DB()
	projectRepo := repository.NewProjectRepository(db)
	tagRepo := repository.NewTagRepository(db)
	return project.NewService(projectRepo, tagRepo)
}

// ListProjects handles GET /projects
// @Summary List projects
// @Description Get a paginated list of projects
// @Tags projects
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param perPage query int false "Items per page" default(20)
// @Success 200 {object} response.SuccessResponse{data=dto.ProjectListResponse}
// @Failure 500 {object} response.SuccessResponse
// @Router /projects [get]
func ListProjects(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	perPage := c.QueryInt("perPage", 20)

	svc := getService()
	result, err := svc.List(c.Context(), page, perPage)
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
// @Summary Get project
// @Description Get a project by ID
// @Tags projects
// @Accept json
// @Produce json
// @Param id path string true "Project ID"
// @Success 200 {object} response.SuccessResponse{data=dto.ProjectResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Router /projects/{id} [get]
func GetProject(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid project ID"))
	}

	svc := getService()
	detail, err := svc.GetDetail(c.Context(), id)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, detail)
}

// CreateProject handles POST /projects
// @Summary Create project
// @Description Create a new project
// @Tags projects
// @Accept json
// @Produce json
// @Param request body dto.CreateProjectRequest true "Project details"
// @Success 201 {object} response.SuccessResponse{data=dto.ProjectResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 401 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /projects [post]
func CreateProject(c *fiber.Ctx) error {
	var req project.CreateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}

	if err := validation.ValidateStruct(req); err != nil {
		return response.Error(c, err)
	}

	svc := getService()
	result, err := svc.Create(c.Context(), req)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, result)
}

// UpdateProject handles PUT /projects/:id
// @Summary Update project
// @Description Update an existing project
// @Tags projects
// @Accept json
// @Produce json
// @Param id path string true "Project ID"
// @Param request body dto.UpdateProjectRequest true "Project update details"
// @Success 200 {object} response.SuccessResponse{data=dto.ProjectResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /projects/{id} [put]
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
	if err := validation.ValidateStruct(req); err != nil {
		return response.Error(c, err)
	}

	svc := getService()
	result, err := svc.Update(c.Context(), id, req)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, result)
}

// DeleteProject handles DELETE /projects/:id
// @Summary Delete project
// @Description Delete a project by ID
// @Tags projects
// @Accept json
// @Produce json
// @Param id path string true "Project ID"
// @Success 200 {object} response.SuccessResponse{data=object{success=boolean}}
// @Failure 400 {object} response.SuccessResponse
// @Failure 401 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /projects/{id} [delete]
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

	svc := getService()
	if err := svc.Delete(c.Context(), id); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{"success": true})
}
