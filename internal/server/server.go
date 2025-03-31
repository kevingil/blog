package server

import (
	"github.com/gofiber/fiber/v2"

	"blog-agent/internal/database"
)

type FiberServer struct {
	*fiber.App

	db database.Service
}

func New() *FiberServer {
	server := &FiberServer{
		App: fiber.New(fiber.Config{
			ServerHeader: "blog-agent",
			AppName:      "blog-agent",
		}),

		db: database.New(),
	}

	return server
}
