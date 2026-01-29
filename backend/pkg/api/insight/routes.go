// Package insight provides HTTP handlers for insight and topic management
package insight

import (
	"backend/pkg/api/middleware"

	"github.com/gofiber/fiber/v2"
)

// Register registers insight routes on the app
func Register(app *fiber.App) {
	// All insight routes require authentication
	ins := app.Group("/insights", middleware.Auth())

	// Static routes must come before dynamic /:id routes
	ins.Get("/", ListInsights)
	ins.Get("/search", SearchInsights)
	ins.Get("/unread-count", GetUnreadCount)

	// Topic routes (static path prefix)
	ins.Get("/topics", ListTopics)
	ins.Post("/topics", CreateTopic)
	ins.Get("/topics/:id", GetTopic)
	ins.Put("/topics/:id", UpdateTopic)
	ins.Delete("/topics/:id", DeleteTopic)

	// Crawled content routes (static path prefix)
	ins.Get("/content/search", SearchCrawledContent)
	ins.Get("/content/recent", GetRecentCrawledContent)

	// Dynamic :id routes must come last
	ins.Get("/:id", GetInsight)
	ins.Delete("/:id", DeleteInsight)
	ins.Post("/:id/read", MarkInsightAsRead)
	ins.Post("/:id/pin", ToggleInsightPinned)
}
