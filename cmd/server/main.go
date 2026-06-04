package main

import (
	"dsr-automation/internal/config"
	env "dsr-automation/pkg/config"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
)

func main() {
	cfg, err := env.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if _, err := config.ConnectDatabase(cfg.DatabaseURL); err != nil {
		log.Fatalf("database: %v", err)
	}

	fmt.Println("connected to database")

	app := fiber.New()

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	fmt.Printf("server listening on :%s\n", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("server: %v", err)
	}
}
