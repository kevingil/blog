package taskrun

import (
	"backend/pkg/api/middleware"

	"github.com/gofiber/fiber/v2"
)

func Register(app *fiber.App) {
	runs := app.Group("/task-runs", middleware.Auth())

	runs.Get("", ListTaskRuns)
	runs.Get("/:id", GetTaskRun)
	runs.Get("/:id/events", ListTaskRunEvents)
}
