package main

import (
	"dsr-automation/internal/database"
	"dsr-automation/internal/handlers"
	"dsr-automation/internal/middleware"
	"dsr-automation/internal/repository"
	"dsr-automation/internal/routes"
	"dsr-automation/internal/services"
	"dsr-automation/pkg/config"
	ghclient "dsr-automation/pkg/github"
	"dsr-automation/pkg/gitlab"
	"dsr-automation/pkg/jwt"
	"dsr-automation/pkg/utils/passwordhashing"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	fmt.Println("connected to database")

	authRepo := repository.NewAuthRepository(db)
	gitRepo := repository.NewGitRepository(db)
	hasher := passwordhashing.NewBcryptHasher(10)
	jwtSvc := jwt.NewService(cfg.JWTSecret, cfg.JWTSecret)
	gitlabClient := gitlab.NewClient()
	githubClient := ghclient.NewClient()

	authService := services.NewAuthService(authRepo, hasher, jwtSvc)
	gitService := services.NewGitIntegrationService(gitRepo, gitlabClient, githubClient)

	authHandler := handlers.NewAuthHandler(authService)
	gitHandler := handlers.NewGitIntegrationHandler(gitService)

	app := fiber.New()
	app.Use(middleware.Logger())
	app.Use(middleware.Recover())
	routes.Setup(app, routes.Dependencies{JWT: jwtSvc}, routes.Handlers{
		Auth: authHandler,
		Git:  gitHandler,
	})

	fmt.Printf("server listening on :%s\n", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("server: %v", err)
	}
}
