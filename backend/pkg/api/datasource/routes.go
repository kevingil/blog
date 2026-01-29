// Package datasource provides HTTP handlers for data source management
package datasource

import (
	"backend/pkg/api/middleware"

	"github.com/gofiber/fiber/v2"
)

// Register registers data source routes on the app
func Register(app *fiber.App) {
	// All data source routes require authentication
	ds := app.Group("/data-sources", middleware.Auth())

	ds.Get("/", ListDataSources)
	ds.Post("/", CreateDataSource)
	ds.Get("/:id", GetDataSource)
	ds.Put("/:id", UpdateDataSource)
	ds.Delete("/:id", DeleteDataSource)
	ds.Post("/:id/crawl", TriggerCrawl)
	ds.Get("/:id/content", GetDataSourceContent)
}
