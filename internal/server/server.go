package server

import (
	"github.com/gofiber/fiber/v2"

	"blog-agent/internal/database"
	user "blog-agent/internal/services/user/services"
)

type FiberServer struct {
	*fiber.App

	db          database.Service
	authService *user.AuthService
}

func New(secretKey string) *FiberServer {
	db := database.New()

	return &FiberServer{
		App: fiber.New(fiber.Config{
			ServerHeader: "blog-agent",
			AppName:      "blog-agent",
		}),
		db:          db,
		authService: user.NewAuthService(db, secretKey),
	}
}
