package dto

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
