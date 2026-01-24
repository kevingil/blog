package handler

import (
	"blog-agent-go/backend/internal/errors"
	"blog-agent-go/backend/internal/response"
	"blog-agent-go/backend/internal/services"
	"blog-agent-go/backend/internal/validation"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func ListPagesHandler(pagesService *services.PagesService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		page := c.QueryInt("page", 1)
		perPage := c.QueryInt("perPage", 20)

		var isPublished *bool
		isPublishedStr := c.Query("isPublished")

		if isPublishedStr != "" {
			val := c.QueryBool("isPublished", true)
			isPublished = &val
		}

		pages, err := pagesService.ListPagesWithPagination(page, perPage, isPublished)
		if err != nil {
			return response.Error(c, err)
		}

		return response.Success(c, pages)
	}
}

func GetPageByIDHandler(pagesService *services.PagesService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid page ID"))
		}

		page, err := pagesService.GetPageByID(id)
		if err != nil {
			return response.Error(c, err)
		}
		if page == nil {
			return response.Error(c, errors.NewNotFoundError("Page"))
		}
		return response.Success(c, page)
	}
}

func CreatePageHandler(pagesService *services.PagesService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req services.PageCreateRequest
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid request body"))
		}
		if err := validation.ValidateStruct(req); err != nil {
			return response.Error(c, err)
		}

		page, err := pagesService.CreatePage(req)
		if err != nil {
			return response.Error(c, err)
		}
		return response.Created(c, page)
	}
}

func UpdatePageHandler(pagesService *services.PagesService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid page ID"))
		}

		var req services.PageUpdateRequest
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid request body"))
		}

		page, err := pagesService.UpdatePage(id, req)
		if err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, page)
	}
}

func DeletePageHandler(pagesService *services.PagesService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid page ID"))
		}

		if err := pagesService.DeletePage(id); err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, fiber.Map{"success": true})
	}
}
