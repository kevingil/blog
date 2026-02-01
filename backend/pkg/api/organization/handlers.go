package organization

import (
	"sync"

	"backend/pkg/api/middleware"
	"backend/pkg/api/response"
	"backend/pkg/api/validation"
	"backend/pkg/core"
	coreOrg "backend/pkg/core/organization"
	"backend/pkg/database"
	"backend/pkg/database/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

var (
	serviceInstance *coreOrg.Service
	serviceOnce     sync.Once
)

// getService returns the organization service instance (lazily initialized)
func getService() *coreOrg.Service {
	serviceOnce.Do(func() {
		db := database.DB()
		orgRepo := repository.NewOrganizationRepository(db)
		accountRepo := repository.NewAccountRepository(db)
		serviceInstance = coreOrg.NewService(orgRepo, accountRepo)
	})
	return serviceInstance
}

// ListOrganizations handles GET /organizations
// @Summary List organizations
// @Description Get a list of all organizations
// @Tags organizations
// @Accept json
// @Produce json
// @Success 200 {object} response.SuccessResponse{data=[]dto.OrganizationResponse}
// @Failure 500 {object} response.SuccessResponse
// @Router /organizations [get]
func ListOrganizations(c *fiber.Ctx) error {
	svc := getService()
	orgs, err := svc.List(c.Context())
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, orgs)
}

// GetOrganization handles GET /organizations/:id
// @Summary Get organization
// @Description Get an organization by ID
// @Tags organizations
// @Accept json
// @Produce json
// @Param id path string true "Organization ID"
// @Success 200 {object} response.SuccessResponse{data=dto.OrganizationResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Router /organizations/{id} [get]
func GetOrganization(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid organization ID"))
	}

	svc := getService()
	org, err := svc.GetByID(c.Context(), id)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, org)
}

// CreateOrganization handles POST /organizations
// @Summary Create organization
// @Description Create a new organization
// @Tags organizations
// @Accept json
// @Produce json
// @Param request body dto.CreateOrganizationRequest true "Organization details"
// @Success 201 {object} response.SuccessResponse{data=dto.OrganizationResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 401 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /organizations [post]
func CreateOrganization(c *fiber.Ctx) error {
	var req coreOrg.CreateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}
	if err := validation.ValidateStruct(req); err != nil {
		return response.Error(c, err)
	}

	svc := getService()
	org, err := svc.Create(c.Context(), req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, org)
}

// UpdateOrganization handles PUT /organizations/:id
// @Summary Update organization
// @Description Update an existing organization
// @Tags organizations
// @Accept json
// @Produce json
// @Param id path string true "Organization ID"
// @Param request body dto.UpdateOrganizationRequest true "Organization update details"
// @Success 200 {object} response.SuccessResponse{data=dto.OrganizationResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /organizations/{id} [put]
func UpdateOrganization(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid organization ID"))
	}

	var req coreOrg.UpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}
	if err := validation.ValidateStruct(req); err != nil {
		return response.Error(c, err)
	}

	svc := getService()
	org, err := svc.Update(c.Context(), id, req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, org)
}

// DeleteOrganization handles DELETE /organizations/:id
// @Summary Delete organization
// @Description Delete an organization by ID
// @Tags organizations
// @Accept json
// @Produce json
// @Param id path string true "Organization ID"
// @Success 200 {object} response.SuccessResponse{data=object{success=boolean}}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /organizations/{id} [delete]
func DeleteOrganization(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid organization ID"))
	}

	svc := getService()
	if err := svc.Delete(c.Context(), id); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"success": true})
}

// JoinOrganization handles POST /organizations/:id/join
// @Summary Join organization
// @Description Join an organization as the current user
// @Tags organizations
// @Accept json
// @Produce json
// @Param id path string true "Organization ID"
// @Success 200 {object} response.SuccessResponse{data=object{success=boolean}}
// @Failure 400 {object} response.SuccessResponse
// @Failure 401 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /organizations/{id}/join [post]
func JoinOrganization(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return response.Error(c, core.UnauthorizedError("Not authenticated"))
	}

	idStr := c.Params("id")
	orgID, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid organization ID"))
	}

	svc := getService()
	if err := svc.JoinOrganization(c.Context(), userID, orgID); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"success": true})
}

// LeaveOrganization handles POST /organizations/leave
// @Summary Leave organization
// @Description Leave the current organization
// @Tags organizations
// @Accept json
// @Produce json
// @Success 200 {object} response.SuccessResponse{data=object{success=boolean}}
// @Failure 401 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /organizations/leave [post]
func LeaveOrganization(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return response.Error(c, core.UnauthorizedError("Not authenticated"))
	}

	svc := getService()
	if err := svc.LeaveOrganization(c.Context(), userID); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"success": true})
}
