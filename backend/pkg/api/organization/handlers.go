package organization

import (
	"backend/pkg/api/middleware"
	"backend/pkg/api/response"
	"backend/pkg/api/validation"
	"backend/pkg/core"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ListOrganizations handles GET /organizations
func ListOrganizations(c *fiber.Ctx) error {
	orgs, err := Organizations().ListOrganizations()
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

	org, err := Organizations().GetOrganizationByID(id)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, org)
}

// CreateOrganization handles POST /organizations
func CreateOrganization(c *fiber.Ctx) error {
	var req OrganizationCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}
	if err := validation.ValidateStruct(req); err != nil {
		return response.Error(c, err)
	}

	org, err := Organizations().CreateOrganization(req)
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

	var req OrganizationUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}

	org, err := Organizations().UpdateOrganization(id, req)
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

	if err := Organizations().DeleteOrganization(id); err != nil {
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

	if err := Organizations().JoinOrganization(userID, orgID); err != nil {
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

	if err := Organizations().LeaveOrganization(userID); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"success": true})
}
