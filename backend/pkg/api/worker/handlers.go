package worker

import (
	"backend/pkg/api/response"
	"backend/pkg/core"
	coreWorker "backend/pkg/core/worker"

	"github.com/gofiber/fiber/v2"
)

// WorkerStatusResponse represents the response for worker status
type WorkerStatusResponse struct {
	Name        string  `json:"name"`
	State       string  `json:"state"`
	Progress    int     `json:"progress"`
	Message     string  `json:"message"`
	StartedAt   *string `json:"started_at,omitempty"`
	CompletedAt *string `json:"completed_at,omitempty"`
	Error       string  `json:"error,omitempty"`
	ItemsTotal  int     `json:"items_total"`
	ItemsDone   int     `json:"items_done"`
}

// AllWorkersStatusResponse represents the response for all workers
type AllWorkersStatusResponse struct {
	Workers   []WorkerStatusResponse `json:"workers"`
	IsRunning bool                   `json:"is_running"`
}

// formatTime formats a time pointer to ISO string or nil
func formatTime(t *string) *string {
	return t
}

// toStatusResponse converts WorkerStatus to response format
func toStatusResponse(status *coreWorker.WorkerStatus) WorkerStatusResponse {
	resp := WorkerStatusResponse{
		Name:       status.Name,
		State:      string(status.State),
		Progress:   status.Progress,
		Message:    status.Message,
		Error:      status.Error,
		ItemsTotal: status.ItemsTotal,
		ItemsDone:  status.ItemsDone,
	}

	if status.StartedAt != nil {
		s := status.StartedAt.Format("2006-01-02T15:04:05Z07:00")
		resp.StartedAt = &s
	}
	if status.CompletedAt != nil {
		s := status.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
		resp.CompletedAt = &s
	}

	return resp
}

// GetAllWorkerStatus handles GET /workers/status
// @Summary Get all workers status
// @Description Get the status of all registered workers
// @Tags workers
// @Accept json
// @Produce json
// @Success 200 {object} response.SuccessResponse{data=AllWorkersStatusResponse}
// @Security BearerAuth
// @Router /workers/status [get]
func GetAllWorkerStatus(c *fiber.Ctx) error {
	manager := coreWorker.GetManager()
	if manager == nil {
		return response.Error(c, core.InternalError("Worker manager not initialized"))
	}

	statusService := coreWorker.GetStatusService()
	statuses := statusService.GetAllStatuses()

	workers := make([]WorkerStatusResponse, 0, len(statuses))
	for _, status := range statuses {
		workers = append(workers, toStatusResponse(&status))
	}

	return response.Success(c, AllWorkersStatusResponse{
		Workers:   workers,
		IsRunning: manager.IsRunning(),
	})
}

// GetWorkerStatus handles GET /workers/:name/status
// @Summary Get worker status
// @Description Get the status of a specific worker
// @Tags workers
// @Accept json
// @Produce json
// @Param name path string true "Worker name"
// @Success 200 {object} response.SuccessResponse{data=WorkerStatusResponse}
// @Failure 404 {object} response.ErrorResponse
// @Security BearerAuth
// @Router /workers/{name}/status [get]
func GetWorkerStatus(c *fiber.Ctx) error {
	name := c.Params("name")

	statusService := coreWorker.GetStatusService()
	status := statusService.GetStatus(name)
	if status == nil {
		return response.Error(c, core.NotFoundError("Worker not found"))
	}

	return response.Success(c, toStatusResponse(status))
}

// RunWorker handles POST /workers/:name/run
// @Summary Run a worker
// @Description Manually trigger a worker to run immediately
// @Tags workers
// @Accept json
// @Produce json
// @Param name path string true "Worker name"
// @Success 200 {object} response.SuccessResponse{data=object{started=bool,message=string}}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Security BearerAuth
// @Router /workers/{name}/run [post]
func RunWorker(c *fiber.Ctx) error {
	name := c.Params("name")

	manager := coreWorker.GetManager()
	if manager == nil {
		return response.Error(c, core.InternalError("Worker manager not initialized"))
	}

	err := manager.RunWorkerNow(name)
	if err != nil {
		switch err {
		case coreWorker.ErrWorkerNotFound:
			return response.Error(c, core.NotFoundError("Worker not found"))
		case coreWorker.ErrWorkerAlreadyRunning:
			return response.Error(c, core.InvalidInputError("Worker is already running"))
		default:
			return response.Error(c, err)
		}
	}

	return response.Success(c, fiber.Map{
		"started": true,
		"message": "Worker started successfully",
	})
}

// StopWorker handles POST /workers/:name/stop
// @Summary Stop a worker
// @Description Stop a running worker
// @Tags workers
// @Accept json
// @Produce json
// @Param name path string true "Worker name"
// @Success 200 {object} response.SuccessResponse{data=object{stopped=bool,message=string}}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Security BearerAuth
// @Router /workers/{name}/stop [post]
func StopWorker(c *fiber.Ctx) error {
	name := c.Params("name")

	manager := coreWorker.GetManager()
	if manager == nil {
		return response.Error(c, core.InternalError("Worker manager not initialized"))
	}

	err := manager.StopWorker(name)
	if err != nil {
		switch err {
		case coreWorker.ErrWorkerNotRunning:
			return response.Error(c, core.InvalidInputError("Worker is not running"))
		default:
			return response.Error(c, err)
		}
	}

	return response.Success(c, fiber.Map{
		"stopped": true,
		"message": "Worker stopped successfully",
	})
}

// GetRunningWorkers handles GET /workers/running
// @Summary Get running workers
// @Description Get list of currently running workers
// @Tags workers
// @Accept json
// @Produce json
// @Success 200 {object} response.SuccessResponse{data=object{workers=[]string}}
// @Security BearerAuth
// @Router /workers/running [get]
func GetRunningWorkers(c *fiber.Ctx) error {
	manager := coreWorker.GetManager()
	if manager == nil {
		return response.Error(c, core.InternalError("Worker manager not initialized"))
	}

	running := manager.GetRunningWorkers()
	return response.Success(c, fiber.Map{
		"workers": running,
	})
}
