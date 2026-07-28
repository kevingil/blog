package middleware

import (
	"github.com/gofiber/fiber/v2"
)

// Security adds security headers to all responses to protect against common vulnerabilities.
// Sets X-Content-Type-Options, X-Frame-Options, X-XSS-Protection, and HSTS (if HTTPS).
func Security() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Prevent MIME type sniffing
		c.Set("X-Content-Type-Options", "nosniff")
		
		// Prevent clickjacking
		c.Set("X-Frame-Options", "DENY")
		
		// Enable XSS protection
		c.Set("X-XSS-Protection", "1; mode=block")
		
		// HSTS - only set if using HTTPS
		if c.Protocol() == "https" {
			c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		
		return c.Next()
	}
}

