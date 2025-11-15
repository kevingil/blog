package controller

import (
	"blog-agent-go/backend/internal/services"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func ListPagesHandler(pagesService *services.PagesService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		fmt.Println("\n=== ListPagesHandler ===")
		fmt.Printf("Request URL: %s\n", c.OriginalURL())
		fmt.Printf("Request Method: %s\n", c.Method())

		page := c.QueryInt("page", 1)
		perPage := c.QueryInt("perPage", 20)
		fmt.Printf("Parsed query params - page: %d, perPage: %d\n", page, perPage)

		var isPublished *bool
		isPublishedStr := c.Query("isPublished")
		fmt.Printf("isPublished query param: '%s'\n", isPublishedStr)

		if isPublishedStr != "" {
			val := c.QueryBool("isPublished", true)
			isPublished = &val
			fmt.Printf("Parsed isPublished: %v\n", *isPublished)
		} else {
			fmt.Println("No isPublished filter")
		}

		fmt.Println("Calling pagesService.ListPagesWithPagination...")
		response, err := pagesService.ListPagesWithPagination(page, perPage, isPublished)
		if err != nil {
			fmt.Printf("ERROR from service: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		fmt.Printf("Success! Returning %d pages\n", len(response.Pages))
		fmt.Println("=== End ListPagesHandler ===")
		return c.JSON(response)
	}
}

func GetPageByIDHandler(pagesService *services.PagesService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid page ID"})
		}

		page, err := pagesService.GetPageByID(id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		if page == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Page not found"})
		}
		return c.JSON(page)
	}
}

func CreatePageHandler(pagesService *services.PagesService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req services.PageCreateRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		page, err := pagesService.CreatePage(req)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusCreated).JSON(page)
	}
}

func UpdatePageHandler(pagesService *services.PagesService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid page ID"})
		}

		var req services.PageUpdateRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		page, err := pagesService.UpdatePage(id, req)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(page)
	}
}

func DeletePageHandler(pagesService *services.PagesService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid page ID"})
		}

		if err := pagesService.DeletePage(id); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"success": true})
	}
}
