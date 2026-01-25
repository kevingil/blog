package handler

import (
	"backend/pkg/database"

	"github.com/gofiber/fiber/v2"
)

// HealthStatus represents the health status response
type HealthStatus struct {
	Status  string            `json:"status"`
	Details map[string]string `json:"details,omitempty"`
}

// LivenessHandler returns the liveness probe handler
// Liveness probe indicates if the application is running
// @Summary		Liveness probe
// @Description	Returns OK if the application is running
// @Tags		health
// @Accept		json
// @Produce		json
// @Success		200	{object}	HealthStatus
// @Router		/health/live [get]
func LivenessHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(HealthStatus{
			Status: "ok",
		})
	}
}

// ReadinessHandler returns the readiness probe handler
// Readiness probe indicates if the application can accept traffic
// @Summary		Readiness probe
// @Description	Returns OK if the application is ready to accept traffic
// @Tags		health
// @Accept		json
// @Produce		json
// @Success		200	{object}	HealthStatus
// @Failure		503	{object}	HealthStatus
// @Router		/health/ready [get]
func ReadinessHandler(dbService database.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		health := dbService.Health()

		status := HealthStatus{
			Status:  "ok",
			Details: make(map[string]string),
		}

		// Check database health
		if health["status"] != "up" {
			status.Status = "degraded"
			status.Details["database"] = health["error"]
			return c.Status(fiber.StatusServiceUnavailable).JSON(status)
		}

		status.Details["database"] = "connected"
		status.Details["database_connections"] = health["open_connections"]

		return c.JSON(status)
	}
}

// FullHealthHandler returns comprehensive health check
// @Summary		Full health check
// @Description	Returns detailed health status of all components
// @Tags		health
// @Accept		json
// @Produce		json
// @Success		200	{object}	HealthStatus
// @Failure		503	{object}	HealthStatus
// @Router		/health [get]
func FullHealthHandler(dbService database.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		health := dbService.Health()

		status := HealthStatus{
			Status:  "ok",
			Details: make(map[string]string),
		}

		// Check database health
		if health["status"] != "up" {
			status.Status = "degraded"
			status.Details["database"] = "down: " + health["error"]
		} else {
			status.Details["database"] = "up"
			status.Details["database_connections"] = health["open_connections"]
			status.Details["database_in_use"] = health["in_use"]
			status.Details["database_idle"] = health["idle"]
		}

		if status.Status != "ok" {
			return c.Status(fiber.StatusServiceUnavailable).JSON(status)
		}

		return c.JSON(status)
	}
}
