package middleware

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
)

// RequestLogger logs incoming HTTP requests with method, path, status code, and duration.
// Useful for debugging and monitoring API usage.
func RequestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		
		// Process request
		err := c.Next()
		
		// Calculate duration
		duration := time.Since(start)
		
		// Log request details
		fmt.Printf("[%s] %s %s - %d - %s\n",
			start.Format("2006-01-02 15:04:05"),
			c.Method(),
			c.Path(),
			c.Response().StatusCode(),
			duration,
		)
		
		return err
	}
}

