package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppEnv            string
	Port              string
	FrontendURL       string
	SessionCookieName string
	SessionTTLHours   int
	CookieSecure      bool
	DBHost            string
	DBPort            string
	DBUser            string
	DBPassword        string
	DBName            string
	DBSSLMode         string
}

func Load() Config {
	appEnv := getEnv("APP_ENV", "development")

	return Config{
		AppEnv:            appEnv,
		Port:              getEnv("SERVER_PORT", "8080"),
		FrontendURL:       getEnv("FRONTEND_URL", "http://localhost:3000"),
		SessionCookieName: getEnv("SESSION_COOKIE_NAME", "idletime_session"),
		SessionTTLHours:   getEnvInt("SESSION_TTL_HOURS", 24*7),
		CookieSecure:      getEnvBool("COOKIE_SECURE", appEnv == "production"),
		DBHost:            getEnv("DB_HOST", "localhost"),
		DBPort:            getEnv("DB_PORT", "5432"),
		DBUser:            getEnv("DB_USER", "idletime"),
		DBPassword:        getEnv("DB_PASSWORD", "idletime"),
		DBName:            getEnv("DB_NAME", "idletime"),
		DBSSLMode:         getEnv("DB_SSLMODE", "disable"),
	}
}

func (c Config) DatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser,
		c.DBPassword,
		c.DBHost,
		c.DBPort,
		c.DBName,
		c.DBSSLMode,
	)
}

func (c Config) SessionTTL() time.Duration {
	return time.Duration(c.SessionTTLHours) * time.Hour
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := getEnv(key, "")
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvBool(key string, fallback bool) bool {
	value := getEnv(key, "")
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
