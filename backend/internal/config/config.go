// Package config loads runtime configuration from the environment.
package config

import (
	"errors"
	"os"
)

const defaultHTTPAddr = ":8080"

// Config holds all runtime settings of the server.
type Config struct {
	HTTPAddr    string
	DatabaseURL string
}

// Load reads the configuration from environment variables.
func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:    getenv("HTTP_ADDR", defaultHTTPAddr),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
