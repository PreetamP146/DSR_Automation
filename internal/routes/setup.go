package routes

import (
	"dsr-automation/internal/handlers"

	"github.com/gofiber/fiber/v2"
)

type Handlers struct {
	Auth handlers.AuthHandler
}

func Setup(app *fiber.App, h Handlers) {
	api := app.Group("/api")

	setupHealthRoutes(api)
	setupAuthRoutes(api, h.Auth)
}
