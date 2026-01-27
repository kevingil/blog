package article

import (
	"backend/pkg/api/middleware"
	"backend/pkg/api/response"
	"backend/pkg/api/validation"
	"backend/pkg/core"
	coreArticle "backend/pkg/core/article"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GenerateArticle handles POST /blog/generate
// @Summary Generate article with AI
// @Description Generate a new blog article using AI based on the provided prompt
// @Tags articles
// @Accept json
// @Produce json
// @Param request body object{prompt=string,title=string,publish=boolean} true "Generation request"
// @Success 200 {object} response.SuccessResponse{data=dto.ArticleResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 401 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /blog/generate [post]
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

	article, err := coreArticle.GenerateArticle(c.Context(), req.Prompt, req.Title, userID, req.Publish)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, article)
}

// UpdateArticle handles POST /blog/articles/:slug/update
// @Summary Update article draft
// @Description Update the draft content of an existing article
// @Tags articles
// @Accept json
// @Produce json
// @Param slug path string true "Article slug"
// @Param request body dto.UpdateArticleRequest true "Update request"
// @Success 200 {object} response.SuccessResponse{data=dto.ArticleResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /blog/articles/{slug}/update [post]
func UpdateArticle(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return response.Error(c, core.InvalidInputError("Article slug is required"))
	}

	articleID, err := coreArticle.GetIDBySlug(c.Context(), slug)
	if err != nil {
		return response.Error(c, err)
	}

	var req coreArticle.UpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}

	if err := validation.ValidateStruct(req); err != nil {
		return response.Error(c, err)
	}

	article, err := coreArticle.Update(c.Context(), articleID, req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, article)
}

// CreateArticle handles POST /blog/articles
// @Summary Create new article
// @Description Create a new blog article
// @Tags articles
// @Accept json
// @Produce json
// @Param request body dto.CreateArticleRequest true "Create request"
// @Success 201 {object} response.SuccessResponse{data=dto.ArticleResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 401 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /blog/articles [post]
func CreateArticle(c *fiber.Ctx) error {
	var req coreArticle.CreateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}

	if err := validation.ValidateStruct(req); err != nil {
		return response.Error(c, err)
	}

	article, err := coreArticle.Create(c.Context(), req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, article)
}

// UpdateArticleWithContext handles PUT /blog/:id/update
// @Summary Update article with AI context
// @Description Update article using AI-powered context-aware editing
// @Tags articles
// @Accept json
// @Produce json
// @Param id path string true "Article ID"
// @Success 200 {object} response.SuccessResponse{data=dto.ArticleResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /blog/{id}/update [put]
func UpdateArticleWithContext(c *fiber.Ctx) error {
	articleIDStr := c.Params("id")
	articleID, err := uuid.Parse(articleIDStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid article ID"))
	}

	article, err := coreArticle.UpdateWithContext(c.Context(), articleID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, article)
}

// GetArticles handles GET /blog/articles
// @Summary List articles
// @Description Get a paginated list of articles with optional filtering
// @Tags articles
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param articlesPerPage query int false "Articles per page" default(6)
// @Param tag query string false "Filter by tag"
// @Param status query string false "Filter by status (published/draft)" default(published)
// @Param sortBy query string false "Sort field"
// @Param sortOrder query string false "Sort order (asc/desc)"
// @Success 200 {object} response.SuccessResponse{data=dto.ArticleListResponse}
// @Failure 500 {object} response.SuccessResponse
// @Router /blog/articles [get]
func GetArticles(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	tag := c.Query("tag", "")
	status := c.Query("status", "published")
	articlesPerPage := c.QueryInt("articlesPerPage", 6)
	sortBy := c.Query("sortBy", "")
	sortOrder := c.Query("sortOrder", "")

	articles, err := coreArticle.List(c.Context(), page, tag, status, articlesPerPage, sortBy, sortOrder)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, articles)
}

// SearchArticles handles GET /blog/articles/search
// @Summary Search articles
// @Description Search articles by query string
// @Tags articles
// @Accept json
// @Produce json
// @Param query query string true "Search query"
// @Param page query int false "Page number" default(1)
// @Param tag query string false "Filter by tag"
// @Param status query string false "Filter by status" default(published)
// @Param sortBy query string false "Sort field"
// @Param sortOrder query string false "Sort order"
// @Success 200 {object} response.SuccessResponse{data=dto.ArticleListResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Router /blog/articles/search [get]
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

	articles, err := coreArticle.Search(c.Context(), query, page, tag, status, sortBy, sortOrder)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, articles)
}

// GetPopularTags handles GET /blog/tags/popular
// @Summary Get popular tags
// @Description Get a list of most used tags
// @Tags articles
// @Accept json
// @Produce json
// @Success 200 {object} response.SuccessResponse{data=object{tags=[]string}}
// @Failure 500 {object} response.SuccessResponse
// @Router /blog/tags/popular [get]
func GetPopularTags(c *fiber.Ctx) error {
	tags, err := coreArticle.GetPopularTags(c.Context())
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"tags": tags})
}

// GetArticleData handles GET /blog/articles/:slug
// @Summary Get article by slug
// @Description Get a single article by its slug
// @Tags articles
// @Accept json
// @Produce json
// @Param slug path string true "Article slug"
// @Success 200 {object} response.SuccessResponse{data=dto.ArticleResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Router /blog/articles/{slug} [get]
func GetArticleData(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return response.Error(c, core.InvalidInputError("Slug is required"))
	}

	data, err := coreArticle.GetBySlug(c.Context(), slug)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, data)
}

// GetRecommendedArticles handles GET /blog/articles/:id/recommended
// @Summary Get recommended articles
// @Description Get articles recommended based on the current article
// @Tags articles
// @Accept json
// @Produce json
// @Param id path string true "Article ID"
// @Success 200 {object} response.SuccessResponse{data=[]dto.ArticleResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Router /blog/articles/{id}/recommended [get]
func GetRecommendedArticles(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid article ID"))
	}

	articles, err := coreArticle.GetRecommended(c.Context(), id)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, articles)
}

// DeleteArticle handles DELETE /blog/articles/:id
// @Summary Delete article
// @Description Permanently delete an article
// @Tags articles
// @Accept json
// @Produce json
// @Param id path string true "Article ID"
// @Success 200 {object} response.SuccessResponse{data=object{success=boolean}}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /blog/articles/{id} [delete]
func DeleteArticle(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid article ID"))
	}

	if err := coreArticle.Delete(c.Context(), id); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"success": true})
}

// PublishArticle handles POST /blog/articles/:slug/publish
// @Summary Publish article
// @Description Publish an article's draft to make it publicly visible
// @Tags articles
// @Accept json
// @Produce json
// @Param slug path string true "Article slug"
// @Success 200 {object} response.SuccessResponse{data=dto.ArticleResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /blog/articles/{slug}/publish [post]
func PublishArticle(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return response.Error(c, core.InvalidInputError("Article slug is required"))
	}

	articleID, err := coreArticle.GetIDBySlug(c.Context(), slug)
	if err != nil {
		return response.Error(c, err)
	}

	article, err := coreArticle.Publish(c.Context(), articleID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, article)
}

// UnpublishArticle handles POST /blog/articles/:slug/unpublish
// @Summary Unpublish article
// @Description Unpublish an article, making it no longer publicly visible
// @Tags articles
// @Accept json
// @Produce json
// @Param slug path string true "Article slug"
// @Success 200 {object} response.SuccessResponse{data=dto.ArticleResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /blog/articles/{slug}/unpublish [post]
func UnpublishArticle(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return response.Error(c, core.InvalidInputError("Article slug is required"))
	}

	articleID, err := coreArticle.GetIDBySlug(c.Context(), slug)
	if err != nil {
		return response.Error(c, err)
	}

	article, err := coreArticle.Unpublish(c.Context(), articleID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, article)
}

// ListVersions handles GET /blog/articles/:slug/versions
// @Summary List article versions
// @Description Get all versions of an article
// @Tags articles
// @Accept json
// @Produce json
// @Param slug path string true "Article slug"
// @Success 200 {object} response.SuccessResponse{data=dto.ArticleVersionListResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /blog/articles/{slug}/versions [get]
func ListVersions(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return response.Error(c, core.InvalidInputError("Article slug is required"))
	}

	articleID, err := coreArticle.GetIDBySlug(c.Context(), slug)
	if err != nil {
		return response.Error(c, err)
	}

	versions, err := coreArticle.ListVersions(c.Context(), articleID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, versions)
}

// GetVersion handles GET /blog/articles/versions/:versionId
// @Summary Get article version
// @Description Get a specific version of an article
// @Tags articles
// @Accept json
// @Produce json
// @Param versionId path string true "Version ID"
// @Success 200 {object} response.SuccessResponse{data=dto.ArticleVersionResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /blog/articles/versions/{versionId} [get]
func GetVersion(c *fiber.Ctx) error {
	versionIDStr := c.Params("versionId")
	versionID, err := uuid.Parse(versionIDStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid version ID"))
	}

	version, err := coreArticle.GetVersion(c.Context(), versionID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, version)
}

// RevertToVersion handles POST /blog/articles/:slug/revert/:versionId
// @Summary Revert article to version
// @Description Revert an article's draft to a previous version
// @Tags articles
// @Accept json
// @Produce json
// @Param slug path string true "Article slug"
// @Param versionId path string true "Version ID to revert to"
// @Success 200 {object} response.SuccessResponse{data=dto.ArticleResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /blog/articles/{slug}/revert/{versionId} [post]
func RevertToVersion(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return response.Error(c, core.InvalidInputError("Article slug is required"))
	}

	articleID, err := coreArticle.GetIDBySlug(c.Context(), slug)
	if err != nil {
		return response.Error(c, err)
	}

	versionIDStr := c.Params("versionId")
	versionID, err := uuid.Parse(versionIDStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid version ID"))
	}

	article, err := coreArticle.RevertToVersion(c.Context(), articleID, versionID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, article)
}
