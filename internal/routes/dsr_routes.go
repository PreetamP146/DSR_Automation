package routes

import (
	"dsr-automation/internal/handlers"
	"dsr-automation/internal/middleware"
	"dsr-automation/pkg/jwt"

	"github.com/gofiber/fiber/v2"
)

func setupDSRRoutes(api fiber.Router, jwtSvc jwt.Service, dsrHandler handlers.DSRHandler) {
	dsr := api.Group("/dsr", middleware.JWT(jwtSvc))

	dsr.Get("/activity", dsrHandler.GetActivity)
	dsr.Post("/generate", dsrHandler.Generate)
	dsr.Get("/", dsrHandler.ListReports)
	dsr.Get("/:id", dsrHandler.GetReport)
}
