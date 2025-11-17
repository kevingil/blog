package controller

import (
	"blog-agent-go/backend/internal/errors"
	"blog-agent-go/backend/internal/response"
	"blog-agent-go/backend/internal/services"
	"blog-agent-go/backend/internal/validation"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func GenerateArticleHandler(blogService *services.ArticleService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			Prompt  string `json:"prompt"`
			Title   string `json:"title"`
			IsDraft bool   `json:"is_draft"`
		}
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid request body"))
		}
		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return response.Error(c, errors.NewUnauthorizedError("User not authenticated"))
		}
		article, err := blogService.GenerateArticle(c.Context(), req.Prompt, req.Title, userID, req.IsDraft)
		if err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, article)
	}
}

func UpdateArticleHandler(blogService *services.ArticleService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		if slug == "" {
			return response.Error(c, errors.NewInvalidInputError("Article slug is required"))
		}
		articleID, err := blogService.GetArticleIDBySlug(slug)
		if err != nil {
			return response.Error(c, err)
		}
		var req services.ArticleUpdateRequest
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid request body"))
		}
		if err := validation.ValidateStruct(req); err != nil {
			return response.Error(c, err)
		}
		article, err := blogService.UpdateArticle(c.Context(), articleID, req)
		if err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, article)
	}
}

func CreateArticleHandler(blogService *services.ArticleService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req services.ArticleCreateRequest
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid request body"))
		}
		if err := validation.ValidateStruct(req); err != nil {
			return response.Error(c, err)
		}
		article, err := blogService.CreateArticle(c.Context(), req)
		if err != nil {
			return response.Error(c, err)
		}
		return response.Created(c, article)
	}
}

func UpdateArticleWithContextHandler(blogService *services.ArticleService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		articleIDStr := c.Params("id")
		articleID, err := uuid.Parse(articleIDStr)
		if err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid article ID"))
		}
		article, err := blogService.UpdateArticleWithContext(c.Context(), articleID)
		if err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, article)
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
		articles, err := blogService.GetArticles(page, tag, status, articlesPerPage, sortBy, sortOrder)
		if err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, articles)
	}
}

func SearchArticlesHandler(blogService *services.ArticleService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		query := c.Query("query")
		if query == "" {
			return response.Error(c, errors.NewInvalidInputError("Query parameter is required"))
		}
		page := c.QueryInt("page", 1)
		tag := c.Query("tag", "")
		status := c.Query("status", "published")
		sortBy := c.Query("sortBy", "")
		sortOrder := c.Query("sortOrder", "")
		articles, err := blogService.SearchArticles(query, page, tag, status, sortBy, sortOrder)
		if err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, articles)
	}
}

func GetPopularTagsHandler(blogService *services.ArticleService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tags, err := blogService.GetPopularTags()
		if err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, fiber.Map{"tags": tags})
	}
}

func GetArticleDataHandler(blogService *services.ArticleService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		if slug == "" {
			return response.Error(c, errors.NewInvalidInputError("Slug is required"))
		}
		data, err := blogService.GetArticleData(slug)
		if err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, data)
	}
}

func GetRecommendedArticlesHandler(blogService *services.ArticleService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid article ID"))
		}
		articles, err := blogService.GetRecommendedArticles(id)
		if err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, articles)
	}
}

func DeleteArticleHandler(blogService *services.ArticleService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			return response.Error(c, errors.NewInvalidInputError("Invalid article ID"))
		}
		if err := blogService.DeleteArticle(id); err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, fiber.Map{"success": true})
	}
}

func GetPageBySlugHandler(pagesService *services.PagesService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		if slug == "" {
			return response.Error(c, errors.NewInvalidInputError("Page slug is required"))
		}
		page, err := pagesService.GetPageBySlug(slug)
		if err != nil {
			return response.Error(c, err)
		}
		if page == nil {
			return response.Error(c, errors.NewNotFoundError("Page"))
		}
		return response.Success(c, page)
	}
}
