package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds runtime configuration loaded from environment variables.
type Config struct {
	DatabaseURL  string
	JWTSecret    string
	Port         string
	OpenAIAPIKey              string
	OpenAIModel               string
	CommitSyncIntervalMinutes int
	CommitSyncLookbackDays    int
	ActivitySyncCronSpec      string
	ActivitySyncLookbackDays  int
}

// Load reads configuration from environment variables.
// It also loads a local .env file when present.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		DatabaseURL:               getEnv("DATABASE_URL"),
		JWTSecret:                 getEnv("JWT_SECRET"),
		Port:                      getEnv("PORT"),
		OpenAIAPIKey:              getEnv("OPENAI_API_KEY"),
		OpenAIModel:               getEnv("OPENAI_MODEL"),
		CommitSyncIntervalMinutes: intEnv("COMMIT_SYNC_INTERVAL_MINUTES", 15),
		CommitSyncLookbackDays:    intEnv("COMMIT_SYNC_LOOKBACK_DAYS", 30),
		ActivitySyncCronSpec:      getEnvWithDefault("ACTIVITY_SYNC_CRON_SPEC", "*/15 * * * *"),
		ActivitySyncLookbackDays:  intEnv("ACTIVITY_SYNC_LOOKBACK_DAYS", 30),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %v", err)
	}

	return cfg, nil
}

func getEnv(key string) string {
	return os.Getenv(key)
}

func getEnvWithDefault(key, defaultValue string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	return value
}

func intEnv(key string, defaultValue int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return defaultValue
	}
	return parsed
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
