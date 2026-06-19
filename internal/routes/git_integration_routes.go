package routes

import (
	"dsr-automation/internal/handlers"
	"dsr-automation/internal/middleware"
	"dsr-automation/pkg/jwt"

	"github.com/gofiber/fiber/v2"
)

func setupGitIntegrationRoutes(api fiber.Router, jwtSvc jwt.Service, gitHandler handlers.GitIntegrationHandler) {
	integrations := api.Group("/integrations", middleware.JWT(jwtSvc))
	git := integrations.Group("/git")

	git.Post("/", gitHandler.Connect)
	git.Get("/", gitHandler.ListIntegrations)
	git.Post("/sync", gitHandler.Sync)
	git.Post("/commits/sync", gitHandler.SyncCommits)
	git.Get("/projects", gitHandler.ListProjects)
	git.Patch("/projects/tracking", gitHandler.UpdateTrackedProjects)
}
