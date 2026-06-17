package routes

import (
	"dsr-automation/internal/handlers"
	"dsr-automation/internal/middleware"
	"dsr-automation/pkg/jwt"

	"github.com/gofiber/fiber/v2"
)

func setupGitIntegrationRoutes(api fiber.Router, jwtSvc jwt.Service, gitHandler handlers.GitIntegrationHandler) {
	integrations := api.Group("/integrations", middleware.JWT(jwtSvc))
	integrations.Post("/git", gitHandler.Connect)
}
