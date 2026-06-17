package routes

import (
	"dsr-automation/internal/handlers"

	"github.com/gofiber/fiber/v2"
)

func setupAuthRoutes(api fiber.Router, authHandler handlers.AuthHandler) {
	auth := api.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)
}
