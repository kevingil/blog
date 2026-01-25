package organization

import (
	"backend/pkg/api/middleware"
	"backend/pkg/api/response"
	"backend/pkg/api/validation"
	"backend/pkg/core"
	coreOrg "backend/pkg/core/organization"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ListOrganizations handles GET /organizations
func ListOrganizations(c *fiber.Ctx) error {
	orgs, err := coreOrg.List(c.Context())
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, orgs)
}

// GetOrganization handles GET /organizations/:id
func GetOrganization(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid organization ID"))
	}

	org, err := coreOrg.GetByID(c.Context(), id)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, org)
}

// CreateOrganization handles POST /organizations
func CreateOrganization(c *fiber.Ctx) error {
	var req coreOrg.CreateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}
	if err := validation.ValidateStruct(req); err != nil {
		return response.Error(c, err)
	}

	org, err := coreOrg.Create(c.Context(), req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, org)
}

// UpdateOrganization handles PUT /organizations/:id
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

	org, err := coreOrg.Update(c.Context(), id, req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, org)
}

// DeleteOrganization handles DELETE /organizations/:id
func DeleteOrganization(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid organization ID"))
	}

	if err := coreOrg.Delete(c.Context(), id); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"success": true})
}

// JoinOrganization handles POST /organizations/:id/join
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

	if err := coreOrg.JoinOrganization(c.Context(), userID, orgID); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"success": true})
}

// LeaveOrganization handles POST /organizations/leave
func LeaveOrganization(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return response.Error(c, core.UnauthorizedError("Not authenticated"))
	}

	if err := coreOrg.LeaveOrganization(c.Context(), userID); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"success": true})
}
