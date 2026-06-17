package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds runtime configuration loaded from environment variables.
type Config struct {
	DatabaseURL string
	JWTSecret   string
	Port        string
}

// Load reads configuration from environment variables.
// It also loads a local .env file when present.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		DatabaseURL: getEnv("DATABASE_URL"),
		JWTSecret:   getEnv("JWT_SECRET"),
		Port:        getEnv("PORT"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %v", err)
	}

	return cfg, nil
}

func getEnv(key string) string {
	return os.Getenv(key)
}

func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return errors.New("DATABASE_URL is required in environment")
	}

	if c.JWTSecret == "" {
		return errors.New("JWT_SECRET is required in environment")
	}

	if c.Port == "" {
		return errors.New("PORT is required in environment")
	}

	return nil
}
