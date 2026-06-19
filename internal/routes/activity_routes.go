package routes

import (
	"dsr-automation/internal/handlers"
	"dsr-automation/internal/middleware"
	"dsr-automation/pkg/jwt"

	"github.com/gofiber/fiber/v2"
)

func setupActivityRoutes(api fiber.Router, jwtSvc jwt.Service, activityHandler handlers.ActivityHandler) {
	activity := api.Group("/activity", middleware.JWT(jwtSvc))

	activity.Get("/", activityHandler.GetActivitySummary)
	activity.Post("/sync", activityHandler.SyncActivity)

	activeRepo := activity.Group("/active-repository")
	activeRepo.Put("/", activityHandler.SetActiveRepository)
	activeRepo.Get("/", activityHandler.GetActiveRepository)

	localGit := activity.Group("/local-git")
	localGit.Post("/repositories", activityHandler.RegisterLocalRepository)
	localGit.Get("/repositories", activityHandler.ListLocalRepositories)
	localGit.Patch("/repositories/tracking", activityHandler.UpdateTrackedLocalRepositories)
	localGit.Delete("/repositories/:id", activityHandler.DeleteLocalRepository)

	planning := activity.Group("/planning")
	planning.Get("/providers", activityHandler.ListPlanningProviders)
	planning.Post("/connect", activityHandler.ConnectPlanning)
	planning.Get("/", activityHandler.ListPlanningIntegrations)
	planning.Get("/integration", activityHandler.GetPlanningIntegration)
	planning.Delete("/", activityHandler.DisconnectPlanning)
}
