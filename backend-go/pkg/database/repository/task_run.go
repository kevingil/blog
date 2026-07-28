package repository

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"backend/pkg/core"
	"backend/pkg/database/models"
	"backend/pkg/types"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type TaskRunFilter struct {
	OrganizationID *uuid.UUID
	UserID         *uuid.UUID
	TaskName       string
	Status         string
	Kind           string
	Limit          int
}

type TaskRunRepository interface {
	CreateRun(ctx context.Context, run *types.TaskRun) error
	UpdateRun(ctx context.Context, run *types.TaskRun) error
	FindRunByID(ctx context.Context, id uuid.UUID) (*types.TaskRun, error)
	ListRuns(ctx context.Context, filter TaskRunFilter) ([]types.TaskRun, error)
	CreateStep(ctx context.Context, step *types.TaskRunStep) error
	UpdateStep(ctx context.Context, step *types.TaskRunStep) error
	FindStepByRunAndKey(ctx context.Context, runID uuid.UUID, stepKey string) (*types.TaskRunStep, error)
	ListStepsByRunID(ctx context.Context, runID uuid.UUID) ([]types.TaskRunStep, error)
	CreateEvent(ctx context.Context, event *types.TaskRunEvent) error
	ListEventsByRunID(ctx context.Context, runID uuid.UUID) ([]types.TaskRunEvent, error)
}

type taskRunRepository struct {
	db *gorm.DB
}

func NewTaskRunRepository(db *gorm.DB) TaskRunRepository {
	return &taskRunRepository{db: db}
}

func marshalJSON(value map[string]interface{}) datatypes.JSON {
	if len(value) == 0 {
		return datatypes.JSON([]byte("{}"))
	}
	data, _ := json.Marshal(value)
	return datatypes.JSON(data)
}

func unmarshalJSON(data datatypes.JSON) map[string]interface{} {
	if len(data) == 0 {
		return map[string]interface{}{}
	}
	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		return map[string]interface{}{}
	}
	return out
}

func taskRunModelToType(model *models.TaskRun) *types.TaskRun {
	return &types.TaskRun{
		ID:                model.ID,
		Kind:              types.TaskRunKind(model.Kind),
		TaskName:          model.TaskName,
		Status:            types.TaskRunStatus(model.Status),
		OrganizationID:    model.OrganizationID,
		UserID:            model.UserID,
		TriggeredByUserID: model.TriggeredByUserID,
		TriggerSource:     model.TriggerSource,
		ParentRunID:       model.ParentRunID,
		Summary:           model.Summary,
		ErrorSummary:      model.ErrorSummary,
		InputPayload:      unmarshalJSON(model.InputPayload),
		OutputSummary:     unmarshalJSON(model.OutputSummary),
		Metrics:           unmarshalJSON(model.Metrics),
		StartedAt:         model.StartedAt,
		CompletedAt:       model.CompletedAt,
		CreatedAt:         model.CreatedAt,
		UpdatedAt:         model.UpdatedAt,
	}
}

func taskRunTypeToModel(run *types.TaskRun) *models.TaskRun {
	return &models.TaskRun{
		ID:                run.ID,
		Kind:              string(run.Kind),
		TaskName:          run.TaskName,
		Status:            string(run.Status),
		OrganizationID:    run.OrganizationID,
		UserID:            run.UserID,
		TriggeredByUserID: run.TriggeredByUserID,
		TriggerSource:     run.TriggerSource,
		ParentRunID:       run.ParentRunID,
		Summary:           run.Summary,
		ErrorSummary:      run.ErrorSummary,
		InputPayload:      marshalJSON(run.InputPayload),
		OutputSummary:     marshalJSON(run.OutputSummary),
		Metrics:           marshalJSON(run.Metrics),
		StartedAt:         run.StartedAt,
		CompletedAt:       run.CompletedAt,
		CreatedAt:         run.CreatedAt,
		UpdatedAt:         run.UpdatedAt,
	}
}

func taskRunStepModelToType(model *models.TaskRunStep) *types.TaskRunStep {
	return &types.TaskRunStep{
		ID:           model.ID,
		TaskRunID:    model.TaskRunID,
		StepKey:      model.StepKey,
		StepName:     model.StepName,
		Status:       types.TaskRunStatus(model.Status),
		Summary:      model.Summary,
		ErrorSummary: model.ErrorSummary,
		Metrics:      unmarshalJSON(model.Metrics),
		StartedAt:    model.StartedAt,
		CompletedAt:  model.CompletedAt,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}
}

func taskRunStepTypeToModel(step *types.TaskRunStep) *models.TaskRunStep {
	return &models.TaskRunStep{
		ID:           step.ID,
		TaskRunID:    step.TaskRunID,
		StepKey:      step.StepKey,
		StepName:     step.StepName,
		Status:       string(step.Status),
		Summary:      step.Summary,
		ErrorSummary: step.ErrorSummary,
		Metrics:      marshalJSON(step.Metrics),
		StartedAt:    step.StartedAt,
		CompletedAt:  step.CompletedAt,
		CreatedAt:    step.CreatedAt,
		UpdatedAt:    step.UpdatedAt,
	}
}

func taskRunEventModelToType(model *models.TaskRunEvent) *types.TaskRunEvent {
	return &types.TaskRunEvent{
		ID:            model.ID,
		TaskRunID:     model.TaskRunID,
		TaskRunStepID: model.TaskRunStepID,
		EventType:     model.EventType,
		Level:         types.TaskRunEventLevel(model.Level),
		Message:       model.Message,
		MetaData:      unmarshalJSON(model.MetaData),
		CreatedAt:     model.CreatedAt,
	}
}

func taskRunEventTypeToModel(event *types.TaskRunEvent) *models.TaskRunEvent {
	return &models.TaskRunEvent{
		ID:            event.ID,
		TaskRunID:     event.TaskRunID,
		TaskRunStepID: event.TaskRunStepID,
		EventType:     event.EventType,
		Level:         string(event.Level),
		Message:       event.Message,
		MetaData:      marshalJSON(event.MetaData),
		CreatedAt:     event.CreatedAt,
	}
}

func (r *taskRunRepository) CreateRun(ctx context.Context, run *types.TaskRun) error {
	model := taskRunTypeToModel(run)
	if model.ID == uuid.Nil {
		model.ID = uuid.New()
		run.ID = model.ID
	}
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *taskRunRepository) UpdateRun(ctx context.Context, run *types.TaskRun) error {
	model := taskRunTypeToModel(run)
	return r.db.WithContext(ctx).Save(model).Error
}

func (r *taskRunRepository) FindRunByID(ctx context.Context, id uuid.UUID) (*types.TaskRun, error) {
	var model models.TaskRun
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return taskRunModelToType(&model), nil
}

func (r *taskRunRepository) ListRuns(ctx context.Context, filter TaskRunFilter) ([]types.TaskRun, error) {
	query := r.db.WithContext(ctx).Model(&models.TaskRun{})
	if filter.OrganizationID != nil {
		query = query.Where("organization_id = ?", *filter.OrganizationID)
	} else if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if taskName := strings.TrimSpace(filter.TaskName); taskName != "" {
		query = query.Where("task_name = ?", taskName)
	}
	if status := strings.TrimSpace(filter.Status); status != "" {
		query = query.Where("status = ?", status)
	}
	if kind := strings.TrimSpace(filter.Kind); kind != "" {
		query = query.Where("kind = ?", kind)
	}

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var modelsOut []models.TaskRun
	if err := query.Order("created_at DESC").Limit(limit).Find(&modelsOut).Error; err != nil {
		return nil, err
	}

	out := make([]types.TaskRun, len(modelsOut))
	for i, model := range modelsOut {
		out[i] = *taskRunModelToType(&model)
	}
	return out, nil
}

func (r *taskRunRepository) CreateStep(ctx context.Context, step *types.TaskRunStep) error {
	model := taskRunStepTypeToModel(step)
	if model.ID == uuid.Nil {
		model.ID = uuid.New()
		step.ID = model.ID
	}
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *taskRunRepository) UpdateStep(ctx context.Context, step *types.TaskRunStep) error {
	model := taskRunStepTypeToModel(step)
	return r.db.WithContext(ctx).Save(model).Error
}

func (r *taskRunRepository) FindStepByRunAndKey(ctx context.Context, runID uuid.UUID, stepKey string) (*types.TaskRunStep, error) {
	var model models.TaskRunStep
	if err := r.db.WithContext(ctx).Where("task_run_id = ? AND step_key = ?", runID, stepKey).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	return taskRunStepModelToType(&model), nil
}

func (r *taskRunRepository) ListStepsByRunID(ctx context.Context, runID uuid.UUID) ([]types.TaskRunStep, error) {
	var modelsOut []models.TaskRunStep
	if err := r.db.WithContext(ctx).Where("task_run_id = ?", runID).Order("created_at ASC").Find(&modelsOut).Error; err != nil {
		return nil, err
	}
	out := make([]types.TaskRunStep, len(modelsOut))
	for i, model := range modelsOut {
		out[i] = *taskRunStepModelToType(&model)
	}
	return out, nil
}

func (r *taskRunRepository) CreateEvent(ctx context.Context, event *types.TaskRunEvent) error {
	model := taskRunEventTypeToModel(event)
	if model.ID == uuid.Nil {
		model.ID = uuid.New()
		event.ID = model.ID
	}
	if model.CreatedAt.IsZero() {
		model.CreatedAt = time.Now()
		event.CreatedAt = model.CreatedAt
	}
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *taskRunRepository) ListEventsByRunID(ctx context.Context, runID uuid.UUID) ([]types.TaskRunEvent, error) {
	var modelsOut []models.TaskRunEvent
	if err := r.db.WithContext(ctx).Where("task_run_id = ?", runID).Order("created_at ASC").Find(&modelsOut).Error; err != nil {
		return nil, err
	}
	out := make([]types.TaskRunEvent, len(modelsOut))
	for i, model := range modelsOut {
		out[i] = *taskRunEventModelToType(&model)
	}
	return out, nil
}
