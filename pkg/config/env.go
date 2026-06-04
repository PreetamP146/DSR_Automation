package config

import (
	"errors"
	"fmt"
	"os"
)

// Config holds runtime configuration loaded from environment variables.
type Config struct {
	DatabaseURL string
	Port        string
}

// Load reads configuration strictly from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL: getEnv("DATABASE_URL"),
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

	if c.Port == "" {
		return errors.New("PORT is required in environment")
	}

	return nil
}
