package config

import (
	"fmt"
	"os"
)

// Config holds the application configuration
type Config struct {
	HTTPAddr string
	DBDriver string
	DBDSN    string
}

// Load loads configuration from environment variables with defaults
func Load() (*Config, error) {
	cfg := &Config{
		HTTPAddr: getEnv("HTTP_ADDR", ":8080"),
		DBDriver: getEnv("DB_DRIVER", "sqlite"),
		DBDSN:    getEnv("DB_DSN", "file:ocpppm.db?_foreign_keys=on"),
	}

	// Validate DB driver
	if cfg.DBDriver != "sqlite" && cfg.DBDriver != "postgres" {
		return nil, fmt.Errorf("invalid DB_DRIVER: %s, must be 'sqlite' or 'postgres'", cfg.DBDriver)
	}

	return cfg, nil
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
