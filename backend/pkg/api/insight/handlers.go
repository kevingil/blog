package insight

import (
	"backend/pkg/api/dto"
	"backend/pkg/api/middleware"
	"backend/pkg/api/response"
	"backend/pkg/api/validation"
	"backend/pkg/core"
	coreInsight "backend/pkg/core/insight"
	"backend/pkg/core/ml"
	"backend/pkg/database"
	"backend/pkg/database/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// getService creates and returns an insight service instance
func getService() *coreInsight.Service {
	db := database.DB()
	return coreInsight.NewService(
		repository.NewInsightRepository(db),
		repository.NewInsightTopicRepository(db),
		repository.NewUserInsightStatusRepository(db),
		repository.NewCrawledContentRepository(db),
		repository.NewContentTopicMatchRepository(db),
		ml.NewEmbeddingService(),
	)
}

// =============================================================================
// Insight Handlers
// =============================================================================

// ListInsights handles GET /insights
// @Summary List insights
// @Description Get a list of all insights with user-specific read/pinned status
// @Tags insights
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param topic_id query string false "Filter by topic ID"
// @Success 200 {object} response.SuccessResponse{data=object{insights=[]dto.InsightWithUserStatus,total=int64}}
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /insights [get]
func ListInsights(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return response.Error(c, err)
	}

	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)
	topicIDStr := c.Query("topic_id")

	svc := getService()

	// For topic filtering, use legacy function then merge with user status
	if topicIDStr != "" {
		topicID, parseErr := uuid.Parse(topicIDStr)
		if parseErr != nil {
			return response.Error(c, core.InvalidInputError("Invalid topic ID"))
		}
		insights, total, err := svc.ListInsightsByTopic(c.Context(), topicID, page, limit)
		if err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, fiber.Map{
			"insights": insights,
			"total":    total,
			"page":     page,
			"limit":    limit,
		})
	}

	// Get insights with user-specific status
	insights, total, err := svc.ListInsightsWithUserStatus(c.Context(), userID, page, limit)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, fiber.Map{
		"insights": insights,
		"total":    total,
		"page":     page,
		"limit":    limit,
	})
}

// GetInsight handles GET /insights/:id
// @Summary Get insight
// @Description Get an insight by ID with its source content
// @Tags insights
// @Accept json
// @Produce json
// @Param id path string true "Insight ID"
// @Success 200 {object} response.SuccessResponse{data=dto.InsightWithSources}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /insights/{id} [get]
func GetInsight(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid insight ID"))
	}

	svc := getService()
	insight, err := svc.GetInsightWithSources(c.Context(), id)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, insight)
}

// MarkInsightAsRead handles POST /insights/:id/read
// @Summary Mark insight as read
// @Description Mark an insight as read for the current user
// @Tags insights
// @Accept json
// @Produce json
// @Param id path string true "Insight ID"
// @Success 200 {object} response.SuccessResponse{data=object{success=boolean}}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /insights/{id}/read [post]
func MarkInsightAsRead(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return response.Error(c, err)
	}

	idStr := c.Params("id")
	insightID, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid insight ID"))
	}

	svc := getService()
	if err := svc.MarkInsightAsReadForUser(c.Context(), userID, insightID); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"success": true})
}

// ToggleInsightPinned handles POST /insights/:id/pin
// @Summary Toggle insight pinned status
// @Description Toggle the pinned status of an insight for the current user
// @Tags insights
// @Accept json
// @Produce json
// @Param id path string true "Insight ID"
// @Success 200 {object} response.SuccessResponse{data=object{success=boolean,is_pinned=boolean}}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /insights/{id}/pin [post]
func ToggleInsightPinned(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return response.Error(c, err)
	}

	idStr := c.Params("id")
	insightID, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid insight ID"))
	}

	svc := getService()
	isPinned, err := svc.ToggleInsightPinnedForUser(c.Context(), userID, insightID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"success": true, "is_pinned": isPinned})
}

// SearchInsights handles GET /insights/search
// @Summary Search insights
// @Description Search insights using semantic similarity
// @Tags insights
// @Accept json
// @Produce json
// @Param q query string true "Search query"
// @Param limit query int false "Max results" default(10)
// @Success 200 {object} response.SuccessResponse{data=[]dto.InsightResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /insights/search [get]
func SearchInsights(c *fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return response.Error(c, core.InvalidInputError("Search query required"))
	}

	limit := c.QueryInt("limit", 10)

	svc := getService()
	orgID := middleware.GetOrgID(c)
	if orgID != nil {
		insights, err := svc.SearchInsightsByOrg(c.Context(), *orgID, query, limit)
		if err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, insights)
	}

	req := dto.InsightSearchRequest{
		Query: query,
		Limit: limit,
	}

	insights, err := svc.SearchInsights(c.Context(), req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, insights)
}

// GetUnreadCount handles GET /insights/unread-count
// @Summary Get unread insight count
// @Description Get the count of unread insights for the current user
// @Tags insights
// @Accept json
// @Produce json
// @Success 200 {object} response.SuccessResponse{data=object{count=int64}}
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /insights/unread-count [get]
func GetUnreadCount(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return response.Error(c, err)
	}

	svc := getService()
	count, err := svc.CountUnreadInsightsForUser(c.Context(), userID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"count": count})
}

// DeleteInsight handles DELETE /insights/:id
// @Summary Delete insight
// @Description Delete an insight by ID
// @Tags insights
// @Accept json
// @Produce json
// @Param id path string true "Insight ID"
// @Success 200 {object} response.SuccessResponse{data=object{success=boolean}}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /insights/{id} [delete]
func DeleteInsight(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid insight ID"))
	}

	svc := getService()
	if err := svc.DeleteInsight(c.Context(), id); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"success": true})
}

// =============================================================================
// Topic Handlers
// =============================================================================

// ListTopics handles GET /insights/topics
// @Summary List topics
// @Description Get a list of all insight topics
// @Tags insights
// @Accept json
// @Produce json
// @Success 200 {object} response.SuccessResponse{data=[]dto.InsightTopicResponse}
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /insights/topics [get]
func ListTopics(c *fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)

	var topics []dto.InsightTopicResponse
	var err error

	svc := getService()
	if orgID != nil {
		topics, err = svc.ListTopics(c.Context(), *orgID)
	} else {
		topics, err = svc.ListAllTopics(c.Context())
	}

	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, topics)
}

// GetTopic handles GET /insights/topics/:id
// @Summary Get topic
// @Description Get a topic by ID
// @Tags insights
// @Accept json
// @Produce json
// @Param id path string true "Topic ID"
// @Success 200 {object} response.SuccessResponse{data=dto.InsightTopicResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /insights/topics/{id} [get]
func GetTopic(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid topic ID"))
	}

	svc := getService()
	topic, err := svc.GetTopicByID(c.Context(), id)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, topic)
}

// CreateTopic handles POST /insights/topics
// @Summary Create topic
// @Description Create a new insight topic
// @Tags insights
// @Accept json
// @Produce json
// @Param request body dto.InsightTopicCreateRequest true "Topic details"
// @Success 201 {object} response.SuccessResponse{data=dto.InsightTopicResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 401 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /insights/topics [post]
func CreateTopic(c *fiber.Ctx) error {
	var req dto.InsightTopicCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}
	if err := validation.ValidateStruct(req); err != nil {
		return response.Error(c, err)
	}

	orgID := middleware.GetOrgID(c)

	svc := getService()
	topic, err := svc.CreateTopic(c.Context(), orgID, req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Created(c, topic)
}

// UpdateTopic handles PUT /insights/topics/:id
// @Summary Update topic
// @Description Update an existing topic
// @Tags insights
// @Accept json
// @Produce json
// @Param id path string true "Topic ID"
// @Param request body dto.InsightTopicUpdateRequest true "Topic update details"
// @Success 200 {object} response.SuccessResponse{data=dto.InsightTopicResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /insights/topics/{id} [put]
func UpdateTopic(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid topic ID"))
	}

	var req dto.InsightTopicUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}

	svc := getService()
	topic, err := svc.UpdateTopic(c.Context(), id, req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, topic)
}

// DeleteTopic handles DELETE /insights/topics/:id
// @Summary Delete topic
// @Description Delete a topic by ID
// @Tags insights
// @Accept json
// @Produce json
// @Param id path string true "Topic ID"
// @Success 200 {object} response.SuccessResponse{data=object{success=boolean}}
// @Failure 400 {object} response.SuccessResponse
// @Failure 404 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /insights/topics/{id} [delete]
func DeleteTopic(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid topic ID"))
	}

	svc := getService()
	if err := svc.DeleteTopic(c.Context(), id); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, fiber.Map{"success": true})
}

// =============================================================================
// Crawled Content Handlers
// =============================================================================

// SearchCrawledContent handles GET /insights/content/search
// @Summary Search crawled content
// @Description Search crawled content using semantic similarity
// @Tags insights
// @Accept json
// @Produce json
// @Param q query string true "Search query"
// @Param limit query int false "Max results" default(10)
// @Success 200 {object} response.SuccessResponse{data=[]dto.CrawledContentResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /insights/content/search [get]
func SearchCrawledContent(c *fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return response.Error(c, core.InvalidInputError("Search query required"))
	}

	limit := c.QueryInt("limit", 10)

	svc := getService()
	orgID := middleware.GetOrgID(c)
	if orgID != nil {
		contents, err := svc.SearchCrawledContentByOrg(c.Context(), *orgID, query, limit)
		if err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, contents)
	}

	contents, err := svc.SearchCrawledContent(c.Context(), query, limit)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, contents)
}

// GetRecentCrawledContent handles GET /insights/content/recent
// @Summary Get recent crawled content
// @Description Get recently crawled content for the organization
// @Tags insights
// @Accept json
// @Produce json
// @Param limit query int false "Max results" default(20)
// @Success 200 {object} response.SuccessResponse{data=[]dto.CrawledContentResponse}
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /insights/content/recent [get]
func GetRecentCrawledContent(c *fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	if orgID == nil {
		return response.Error(c, core.UnauthorizedError("Organization required"))
	}

	limit := c.QueryInt("limit", 20)

	svc := getService()
	contents, err := svc.GetRecentCrawledContent(c.Context(), *orgID, limit)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, contents)
}
