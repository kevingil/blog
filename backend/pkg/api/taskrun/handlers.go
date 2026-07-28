package taskrun

import (
	"time"

	"backend/pkg/api/dto"
	"backend/pkg/api/middleware"
	"backend/pkg/api/response"
	"backend/pkg/core"
	"backend/pkg/core/taskrun"
	"backend/pkg/database"
	"backend/pkg/database/repository"
	"backend/pkg/types"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func newTaskRunService() *taskrun.Service {
	return taskrun.NewService(repository.NewTaskRunRepository(database.DB()))
}

func toTaskRunResponse(run types.TaskRun) dto.TaskRunResponse {
	resp := dto.TaskRunResponse{
		ID:            run.ID.String(),
		Kind:          string(run.Kind),
		TaskName:      run.TaskName,
		Status:        string(run.Status),
		TriggerSource: run.TriggerSource,
		Summary:       run.Summary,
		ErrorSummary:  run.ErrorSummary,
		OutputSummary: run.OutputSummary,
		Metrics:       run.Metrics,
	}
	if run.ParentRunID != nil {
		parentID := run.ParentRunID.String()
		resp.ParentRunID = &parentID
	}
	if run.StartedAt != nil {
		value := run.StartedAt.Format(time.RFC3339)
		resp.StartedAt = &value
	}
	if run.CompletedAt != nil {
		value := run.CompletedAt.Format(time.RFC3339)
		resp.CompletedAt = &value
	}
	if run.StartedAt != nil && run.CompletedAt != nil {
		duration := run.CompletedAt.Sub(*run.StartedAt).Milliseconds()
		resp.DurationMS = &duration
	}
	return resp
}

func toTaskRunStepResponse(step types.TaskRunStep) dto.TaskRunStepResponse {
	resp := dto.TaskRunStepResponse{
		ID:           step.ID.String(),
		StepKey:      step.StepKey,
		StepName:     step.StepName,
		Status:       string(step.Status),
		Summary:      step.Summary,
		ErrorSummary: step.ErrorSummary,
		Metrics:      step.Metrics,
	}
	if step.StartedAt != nil {
		value := step.StartedAt.Format(time.RFC3339)
		resp.StartedAt = &value
	}
	if step.CompletedAt != nil {
		value := step.CompletedAt.Format(time.RFC3339)
		resp.CompletedAt = &value
	}
	return resp
}

func toTaskRunEventResponse(event types.TaskRunEvent, stepKey *string) dto.TaskRunEventResponse {
	return dto.TaskRunEventResponse{
		ID:        event.ID.String(),
		EventType: event.EventType,
		Level:     string(event.Level),
		Message:   event.Message,
		CreatedAt: event.CreatedAt.Format(time.RFC3339),
		StepKey:   stepKey,
		MetaData:  event.MetaData,
	}
}

func runFilterFromRequest(c *fiber.Ctx) (repository.TaskRunFilter, error) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return repository.TaskRunFilter{}, core.UnauthorizedError("User not found")
	}
	filter := repository.TaskRunFilter{
		UserID:   &userID,
		TaskName: c.Query("task_name"),
		Status:   c.Query("status"),
		Kind:     c.Query("kind"),
		Limit:    c.QueryInt("limit", 50),
	}
	orgID := middleware.GetOrgID(c)
	if orgID != nil {
		filter.OrganizationID = orgID
		filter.UserID = nil
	}
	return filter, nil
}

func ensureRunAccess(c *fiber.Ctx, run *types.TaskRun) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return core.UnauthorizedError("User not found")
	}
	orgID := middleware.GetOrgID(c)

	if orgID != nil {
		if run.OrganizationID == nil || *run.OrganizationID != *orgID {
			return core.NotFoundError("Task run not found")
		}
		return nil
	}

	if run.UserID == nil || *run.UserID != userID {
		return core.NotFoundError("Task run not found")
	}
	return nil
}

func ListTaskRuns(c *fiber.Ctx) error {
	filter, err := runFilterFromRequest(c)
	if err != nil {
		return response.Error(c, err)
	}

	runs, err := newTaskRunService().ListRuns(c.Context(), filter)
	if err != nil {
		return response.Error(c, err)
	}

	items := make([]dto.TaskRunResponse, len(runs))
	for i, run := range runs {
		items[i] = toTaskRunResponse(run)
	}

	return response.Success(c, dto.TaskRunListResponse{Runs: items})
}

func GetTaskRun(c *fiber.Ctx) error {
	runID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid task run ID"))
	}

	service := newTaskRunService()
	run, err := service.GetRun(c.Context(), runID)
	if err != nil {
		return response.Error(c, err)
	}
	if err := ensureRunAccess(c, run); err != nil {
		return response.Error(c, err)
	}

	steps, err := service.ListStepsByRunID(c.Context(), runID)
	if err != nil {
		return response.Error(c, err)
	}
	stepKeysByID := map[uuid.UUID]string{}
	stepResponses := make([]dto.TaskRunStepResponse, len(steps))
	for i, step := range steps {
		stepKeysByID[step.ID] = step.StepKey
		stepResponses[i] = toTaskRunStepResponse(step)
	}

	events, err := service.ListEventsByRunID(c.Context(), runID)
	if err != nil {
		return response.Error(c, err)
	}
	eventResponses := make([]dto.TaskRunEventResponse, len(events))
	for i, event := range events {
		var stepKey *string
		if event.TaskRunStepID != nil {
			if key, ok := stepKeysByID[*event.TaskRunStepID]; ok {
				stepKey = &key
			}
		}
		eventResponses[i] = toTaskRunEventResponse(event, stepKey)
	}

	return response.Success(c, dto.TaskRunDetailResponse{
		Run:    toTaskRunResponse(*run),
		Steps:  stepResponses,
		Events: eventResponses,
	})
}

func ListTaskRunEvents(c *fiber.Ctx) error {
	runID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, core.InvalidInputError("Invalid task run ID"))
	}

	service := newTaskRunService()
	run, err := service.GetRun(c.Context(), runID)
	if err != nil {
		return response.Error(c, err)
	}
	if err := ensureRunAccess(c, run); err != nil {
		return response.Error(c, err)
	}
	steps, err := service.ListStepsByRunID(c.Context(), runID)
	if err != nil {
		return response.Error(c, err)
	}
	stepKeysByID := map[uuid.UUID]string{}
	for _, step := range steps {
		stepKeysByID[step.ID] = step.StepKey
	}
	events, err := service.ListEventsByRunID(c.Context(), runID)
	if err != nil {
		return response.Error(c, err)
	}
	items := make([]dto.TaskRunEventResponse, len(events))
	for i, event := range events {
		var stepKey *string
		if event.TaskRunStepID != nil {
			if key, ok := stepKeysByID[*event.TaskRunStepID]; ok {
				stepKey = &key
			}
		}
		items[i] = toTaskRunEventResponse(event, stepKey)
	}
	return response.Success(c, fiber.Map{"events": items})
}
