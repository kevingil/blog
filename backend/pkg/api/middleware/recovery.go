package middleware

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// Recovery recovers from panics in request handlers and returns a 500 Internal Server Error.
// Prevents the entire server from crashing when a panic occurs in a handler.
func Recovery() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("PANIC RECOVERED: %v\n", r)
				c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}
		}()
		return c.Next()
	}
}
