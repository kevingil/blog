package article

import (
	"backend/pkg/api/middleware"
	"backend/pkg/api/response"
	"backend/pkg/api/services"
	"backend/pkg/api/validation"
	"backend/pkg/core"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GenerateArticle handles POST /blog/generate
func GenerateArticle(c *fiber.Ctx) error {
	var req struct {
		Prompt  string `json:"prompt"`
		Title   string `json:"title"`
		Publish bool   `json:"publish"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}

	userID, err := middleware.GetUserID(c)
	if err != nil {
		return response.Error(c, core.UnauthorizedError("User not authenticated"))
	}

	article, err := services.Articles().GenerateArticle(c.Context(), req.Prompt, req.Title, userID, req.Publish)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, article)
}

// UpdateArticle handles POST /blog/articles/:slug/update
func UpdateArticle(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return response.Error(c, core.InvalidInputError("Article slug is required"))
	}

	articleID, err := services.Articles().GetArticleIDBySlug(slug)
	if err != nil {
		return response.Error(c, err)
	}

	var req services.ArticleUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}

	if err := validation.ValidateStruct(req); err != nil {
		return response.Error(c, err)
	}

	article, err := services.Articles().UpdateArticle(c.Context(), articleID, req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, article)
}

// CreateArticle handles POST /blog/articles
func CreateArticle(c *fiber.Ctx) error {
	var req services.ArticleCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}

	if err := validation.ValidateStruct(req); err != nil {
		return response.Error(c, err)
	}

	article, err := services.Articles().CreateArticle(c.Context(), req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, article)
}

// UpdateArticleWithContext handles PUT /blog/:id/update
func UpdateArticleWithContext(c *fiber.Ctx) error {
	articleIDStr := c.Params("id")
	articleID, err := uuid.Parse(articleIDStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid article ID"))
	}

	article, err := services.Articles().UpdateArticleWithContext(c.Context(), articleID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, article)
}

// GetArticles handles GET /blog/articles
func GetArticles(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	tag := c.Query("tag", "")
	status := c.Query("status", "published")
	articlesPerPage := c.QueryInt("articlesPerPage", 6)
	sortBy := c.Query("sortBy", "")
	sortOrder := c.Query("sortOrder", "")

	articles, err := services.Articles().GetArticles(page, tag, status, articlesPerPage, sortBy, sortOrder)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, articles)
}

// SearchArticles handles GET /blog/articles/search
func SearchArticles(c *fiber.Ctx) error {
	query := c.Query("query")
	if query == "" {
		return response.Error(c, core.InvalidInputError("Query parameter is required"))
	}

	page := c.QueryInt("page", 1)
	tag := c.Query("tag", "")
	status := c.Query("status", "published")
	sortBy := c.Query("sortBy", "")
	sortOrder := c.Query("sortOrder", "")

	articles, err := services.Articles().SearchArticles(query, page, tag, status, sortBy, sortOrder)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, articles)
}

// GetPopularTags handles GET /blog/tags/popular
func GetPopularTags(c *fiber.Ctx) error {
	tags, err := services.Articles().GetPopularTags()
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"tags": tags})
}

// GetArticleData handles GET /blog/articles/:slug
func GetArticleData(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return response.Error(c, core.InvalidInputError("Slug is required"))
	}

	data, err := services.Articles().GetArticleData(slug)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, data)
}

// GetRecommendedArticles handles GET /blog/articles/:id/recommended
func GetRecommendedArticles(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid article ID"))
	}

	articles, err := services.Articles().GetRecommendedArticles(id)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, articles)
}

// DeleteArticle handles DELETE /blog/articles/:id
func DeleteArticle(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid article ID"))
	}

	if err := services.Articles().DeleteArticle(id); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"success": true})
}

// PublishArticle handles POST /blog/articles/:slug/publish
func PublishArticle(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return response.Error(c, core.InvalidInputError("Article slug is required"))
	}

	articleID, err := services.Articles().GetArticleIDBySlug(slug)
	if err != nil {
		return response.Error(c, err)
	}

	article, err := services.Articles().PublishArticle(c.Context(), articleID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, article)
}

// UnpublishArticle handles POST /blog/articles/:slug/unpublish
func UnpublishArticle(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return response.Error(c, core.InvalidInputError("Article slug is required"))
	}

	articleID, err := services.Articles().GetArticleIDBySlug(slug)
	if err != nil {
		return response.Error(c, err)
	}

	article, err := services.Articles().UnpublishArticle(c.Context(), articleID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, article)
}

// ListVersions handles GET /blog/articles/:slug/versions
func ListVersions(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return response.Error(c, core.InvalidInputError("Article slug is required"))
	}

	articleID, err := services.Articles().GetArticleIDBySlug(slug)
	if err != nil {
		return response.Error(c, err)
	}

	versions, err := services.Articles().ListVersions(c.Context(), articleID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, versions)
}

// GetVersion handles GET /blog/articles/versions/:versionId
func GetVersion(c *fiber.Ctx) error {
	versionIDStr := c.Params("versionId")
	versionID, err := uuid.Parse(versionIDStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid version ID"))
	}

	version, err := services.Articles().GetVersion(c.Context(), versionID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, version)
}

// RevertToVersion handles POST /blog/articles/:slug/revert/:versionId
func RevertToVersion(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return response.Error(c, core.InvalidInputError("Article slug is required"))
	}

	articleID, err := services.Articles().GetArticleIDBySlug(slug)
	if err != nil {
		return response.Error(c, err)
	}

	versionIDStr := c.Params("versionId")
	versionID, err := uuid.Parse(versionIDStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid version ID"))
	}

	article, err := services.Articles().RevertToVersion(c.Context(), articleID, versionID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, article)
}
