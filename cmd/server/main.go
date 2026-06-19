package main

import (
	"context"
	"dsr-automation/internal/database"
	"dsr-automation/internal/handlers"
	"dsr-automation/internal/middleware"
	"dsr-automation/internal/repository"
	"dsr-automation/internal/routes"
	"dsr-automation/internal/services"
	"dsr-automation/internal/worker"
	"dsr-automation/pkg/ai"
	"dsr-automation/pkg/config"
	ghclient "dsr-automation/pkg/github"
	"dsr-automation/pkg/gitlab"
	jiraclient "dsr-automation/pkg/jira"
	"dsr-automation/pkg/localgit"
	jiraprovider "dsr-automation/pkg/planning/jira"
	"dsr-automation/pkg/planning"
	trelloprovider "dsr-automation/pkg/planning/trello"
	"dsr-automation/pkg/jwt"
	"dsr-automation/pkg/utils/passwordhashing"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
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
	commitRepo := repository.NewCommitRepository(db)
	dsrRepo := repository.NewDSRRepository(db)
	activityRepo := repository.NewActivityRepository(db)
	hasher := passwordhashing.NewBcryptHasher(10)
	jwtSvc := jwt.NewService(cfg.JWTSecret, cfg.JWTSecret)
	gitlabClient := gitlab.NewClient()
	githubClient := ghclient.NewClient()
	jiraClient := jiraclient.NewClient()
	localGitScanner := localgit.NewScanner()
	summarizer := ai.NewSummarizer(ai.Config{
		OpenAIAPIKey: cfg.OpenAIAPIKey,
		OpenAIModel:  cfg.OpenAIModel,
	})

	authService := services.NewAuthService(authRepo, hasher, jwtSvc)
	gitService := services.NewGitIntegrationService(gitRepo, gitlabClient, githubClient)
	commitSyncService := services.NewCommitSyncService(
		gitRepo,
		commitRepo,
		githubClient,
		gitlabClient,
		cfg.CommitSyncLookbackDays,
	)
	dsrService := services.NewDSRService(dsrRepo, gitRepo, commitRepo, activityRepo, summarizer)
	localGitService := services.NewLocalGitService(activityRepo, localGitScanner)
	localGitSyncService := services.NewLocalGitSyncService(activityRepo, localGitScanner, cfg.ActivitySyncLookbackDays)
	planningRegistry := planning.NewRegistry(
		jiraprovider.NewProvider(jiraClient),
		trelloprovider.NewProvider(),
	)
	planningIntegrationService := services.NewPlanningIntegrationService(activityRepo, planningRegistry)
	planningSyncService := services.NewPlanningSyncService(activityRepo, planningRegistry, cfg.ActivitySyncLookbackDays)
	activityTrackerService := services.NewActivityTrackerService(activityRepo, localGitSyncService, planningSyncService)

	authHandler := handlers.NewAuthHandler(authService)
	gitHandler := handlers.NewGitIntegrationHandler(gitService, commitSyncService)
	dsrHandler := handlers.NewDSRHandler(dsrService)
	activityHandler := handlers.NewActivityHandler(localGitService, planningIntegrationService, activityTrackerService)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go worker.StartCommitSyncWorker(
		ctx,
		commitSyncService,
		time.Duration(cfg.CommitSyncIntervalMinutes)*time.Minute,
	)
	go worker.StartActivitySyncCron(ctx, activityTrackerService, cfg.ActivitySyncCronSpec)

	app := fiber.New()
	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:3000,http://localhost:3001",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
	}))
	app.Use(middleware.Logger())
	app.Use(middleware.Recover())
	routes.Setup(app, routes.Dependencies{JWT: jwtSvc}, routes.Handlers{
		Auth:     authHandler,
		Git:      gitHandler,
		DSR:      dsrHandler,
		Activity: activityHandler,
	})

	fmt.Printf("server listening on :%s\n", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("server: %v", err)
	}
}
