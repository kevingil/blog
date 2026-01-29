package insight

import (
	"backend/pkg/api/middleware"
	"backend/pkg/api/response"
	"backend/pkg/api/validation"
	"backend/pkg/core"
	coreInsight "backend/pkg/core/insight"
	"backend/pkg/types"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

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
// @Success 200 {object} response.SuccessResponse{data=object{insights=[]types.InsightWithUserStatus,total=int64}}
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

	// For topic filtering, use legacy function then merge with user status
	if topicIDStr != "" {
		topicID, parseErr := uuid.Parse(topicIDStr)
		if parseErr != nil {
			return response.Error(c, core.InvalidInputError("Invalid topic ID"))
		}
		insights, total, err := coreInsight.ListInsightsByTopic(c.Context(), topicID, page, limit)
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
	insights, total, err := coreInsight.ListInsightsWithUserStatus(c.Context(), userID, page, limit)
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
// @Success 200 {object} response.SuccessResponse{data=types.InsightWithSources}
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

	insight, err := coreInsight.GetInsightWithSources(c.Context(), id)
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

	if err := coreInsight.MarkInsightAsReadForUser(c.Context(), userID, insightID); err != nil {
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

	isPinned, err := coreInsight.ToggleInsightPinnedForUser(c.Context(), userID, insightID)
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
// @Success 200 {object} response.SuccessResponse{data=[]types.InsightResponse}
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

	orgID := middleware.GetOrgID(c)
	if orgID != nil {
		insights, err := coreInsight.SearchInsightsByOrg(c.Context(), *orgID, query, limit)
		if err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, insights)
	}

	req := types.InsightSearchRequest{
		Query: query,
		Limit: limit,
	}

	insights, err := coreInsight.SearchInsights(c.Context(), req)
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

	count, err := coreInsight.CountUnreadInsightsForUser(c.Context(), userID)
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

	if err := coreInsight.DeleteInsight(c.Context(), id); err != nil {
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
// @Success 200 {object} response.SuccessResponse{data=[]types.InsightTopicResponse}
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /insights/topics [get]
func ListTopics(c *fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)

	var topics []types.InsightTopicResponse
	var err error

	if orgID != nil {
		topics, err = coreInsight.ListTopics(c.Context(), *orgID)
	} else {
		topics, err = coreInsight.ListAllTopics(c.Context())
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
// @Success 200 {object} response.SuccessResponse{data=types.InsightTopicResponse}
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

	topic, err := coreInsight.GetTopicByID(c.Context(), id)
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
// @Param request body types.InsightTopicCreateRequest true "Topic details"
// @Success 201 {object} response.SuccessResponse{data=types.InsightTopicResponse}
// @Failure 400 {object} response.SuccessResponse
// @Failure 401 {object} response.SuccessResponse
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /insights/topics [post]
func CreateTopic(c *fiber.Ctx) error {
	var req types.InsightTopicCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}
	if err := validation.ValidateStruct(req); err != nil {
		return response.Error(c, err)
	}

	orgID := middleware.GetOrgID(c)

	topic, err := coreInsight.CreateTopic(c.Context(), orgID, req)
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
// @Param request body types.InsightTopicUpdateRequest true "Topic update details"
// @Success 200 {object} response.SuccessResponse{data=types.InsightTopicResponse}
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

	var req types.InsightTopicUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, core.InvalidInputError("Invalid request body"))
	}

	topic, err := coreInsight.UpdateTopic(c.Context(), id, req)
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

	if err := coreInsight.DeleteTopic(c.Context(), id); err != nil {
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
// @Success 200 {object} response.SuccessResponse{data=[]types.CrawledContentResponse}
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

	orgID := middleware.GetOrgID(c)
	if orgID != nil {
		contents, err := coreInsight.SearchCrawledContentByOrg(c.Context(), *orgID, query, limit)
		if err != nil {
			return response.Error(c, err)
		}
		return response.Success(c, contents)
	}

	contents, err := coreInsight.SearchCrawledContent(c.Context(), query, limit)
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
// @Success 200 {object} response.SuccessResponse{data=[]types.CrawledContentResponse}
// @Failure 500 {object} response.SuccessResponse
// @Security BearerAuth
// @Router /insights/content/recent [get]
func GetRecentCrawledContent(c *fiber.Ctx) error {
	orgID := middleware.GetOrgID(c)
	if orgID == nil {
		return response.Error(c, core.UnauthorizedError("Organization required"))
	}

	limit := c.QueryInt("limit", 20)

	contents, err := coreInsight.GetRecentCrawledContent(c.Context(), *orgID, limit)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, contents)
}
