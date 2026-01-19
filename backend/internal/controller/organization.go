package controller

import (
	"blog-agent-go/backend/internal/errors"
	"blog-agent-go/backend/internal/middleware"
	"blog-agent-go/backend/internal/response"
	"blog-agent-go/backend/internal/services"
	"blog-agent-go/backend/internal/validation"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ListOrganizationsHandler returns all organizations
func ListOrganizationsHandler(orgService *services.OrganizationService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		orgs, err := orgService.ListOrganizations()
		if err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, orgs)
	}
}

// GetOrganizationHandler returns an organization by ID
func GetOrganizationHandler(orgService *services.OrganizationService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid organization ID"))
		}

		org, err := orgService.GetOrganizationByID(id)
		if err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, org)
	}
}

// CreateOrganizationHandler creates a new organization
func CreateOrganizationHandler(orgService *services.OrganizationService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req services.OrganizationCreateRequest
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid request body"))
		}
		if err := validation.ValidateStruct(req); err != nil {
			return response.Error(c, err)
		}

		org, err := orgService.CreateOrganization(req)
		if err != nil {
			return response.Error(c, err)
		}
		return response.Created(c, org)
	}
}

// UpdateOrganizationHandler updates an organization
func UpdateOrganizationHandler(orgService *services.OrganizationService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid organization ID"))
		}

		var req services.OrganizationUpdateRequest
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid request body"))
		}

		org, err := orgService.UpdateOrganization(id, req)
		if err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, org)
	}
}

// DeleteOrganizationHandler deletes an organization
func DeleteOrganizationHandler(orgService *services.OrganizationService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid organization ID"))
		}

		if err := orgService.DeleteOrganization(id); err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, fiber.Map{"success": true})
	}
}

// JoinOrganizationHandler allows a user to join an organization
func JoinOrganizationHandler(orgService *services.OrganizationService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, err := middleware.GetUserID(c)
		if err != nil {
			return response.Error(c, errors.NewUnauthorizedError("Not authenticated"))
		}

		idStr := c.Params("id")
		orgID, err := uuid.Parse(idStr)
		if err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid organization ID"))
		}

		if err := orgService.JoinOrganization(userID, orgID); err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, fiber.Map{"success": true})
	}
}

// LeaveOrganizationHandler allows a user to leave their organization
func LeaveOrganizationHandler(orgService *services.OrganizationService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, err := middleware.GetUserID(c)
		if err != nil {
			return response.Error(c, errors.NewUnauthorizedError("Not authenticated"))
		}

		if err := orgService.LeaveOrganization(userID); err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, fiber.Map{"success": true})
	}
}
