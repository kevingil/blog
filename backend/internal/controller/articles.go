package controller

import (
	"blog-agent-go/backend/internal/services"
	"database/sql"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GenerateArticleHandler(blogService *services.ArticleService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			Prompt  string `json:"prompt"`
			Title   string `json:"title"`
			IsDraft bool   `json:"is_draft"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		userIDStr := c.Locals("userID").(string)
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
		}
		article, err := blogService.GenerateArticle(c.Context(), req.Prompt, req.Title, userID, req.IsDraft)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(article)
	}
}

func UpdateArticleHandler(blogService *services.ArticleService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		if slug == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Article slug is required"})
		}
		articleID, err := blogService.GetArticleIDBySlug(slug)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Article not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to find article"})
		}
		var req services.ArticleUpdateRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		article, err := blogService.UpdateArticle(c.Context(), articleID, req)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(article)
	}
}

func CreateArticleHandler(blogService *services.ArticleService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req services.ArticleCreateRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		article, err := blogService.CreateArticle(c.Context(), req)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(article)
	}
}

func UpdateArticleWithContextHandler(blogService *services.ArticleService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		articleIDStr := c.Params("id")
		articleID, err := uuid.Parse(articleIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid article ID"})
		}
		article, err := blogService.UpdateArticleWithContext(c.Context(), articleID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(article)
	}
}

func GetArticlesHandler(blogService *services.ArticleService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		page := c.QueryInt("page", 1)
		tag := c.Query("tag", "")
		status := c.Query("status", "published")
		articlesPerPage := c.QueryInt("articlesPerPage", 6)
		sortBy := c.Query("sortBy", "")
		sortOrder := c.Query("sortOrder", "")
		response, err := blogService.GetArticles(page, tag, status, articlesPerPage, sortBy, sortOrder)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(response)
	}
}

func SearchArticlesHandler(blogService *services.ArticleService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		query := c.Query("query")
		if query == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Query parameter is required"})
		}
		page := c.QueryInt("page", 1)
		tag := c.Query("tag", "")
		status := c.Query("status", "published")
		sortBy := c.Query("sortBy", "")
		sortOrder := c.Query("sortOrder", "")
		response, err := blogService.SearchArticles(query, page, tag, status, sortBy, sortOrder)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(response)
	}
}

func GetPopularTagsHandler(blogService *services.ArticleService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tags, err := blogService.GetPopularTags()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"tags": tags})
	}
}

func GetArticleDataHandler(blogService *services.ArticleService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		if slug == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Slug is required"})
		}
		data, err := blogService.GetArticleData(slug)
		if err != nil {
			if err == sql.ErrNoRows {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Article not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(data)
	}
}

func GetRecommendedArticlesHandler(blogService *services.ArticleService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid article ID"})
		}
		articles, err := blogService.GetRecommendedArticles(id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(articles)
	}
}

func DeleteArticleHandler(blogService *services.ArticleService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid article ID"})
		}
		if err := blogService.DeleteArticle(id); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"success": true})
	}
}

func GetPageBySlugHandler(pagesService *services.PagesService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		if slug == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Page slug is required"})
		}
		page, err := pagesService.GetPageBySlug(slug)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		if page == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Page not found"})
		}
		return c.JSON(page)
	}
}
