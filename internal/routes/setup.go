package routes

import (
	"dsr-automation/internal/handlers"
	"dsr-automation/pkg/jwt"

	"github.com/gofiber/fiber/v2"
)

type Handlers struct {
	Auth     handlers.AuthHandler
	Git      handlers.GitIntegrationHandler
	DSR      handlers.DSRHandler
	Activity handlers.ActivityHandler
}

type Dependencies struct {
	JWT jwt.Service
}

func Setup(app *fiber.App, deps Dependencies, h Handlers) {
	api := app.Group("/api")

	setupHealthRoutes(api)
	setupAuthRoutes(api, h.Auth)
	setupGitIntegrationRoutes(api, deps.JWT, h.Git)
	setupDSRRoutes(api, deps.JWT, h.DSR)
	setupActivityRoutes(api, deps.JWT, h.Activity)
}
