// Package config loads runtime configuration from the environment.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/schellhas/idletime/backend/internal/mail"
)

const (
	defaultHTTPAddr   = ":8080"
	defaultAppBaseURL = "http://localhost:3000"
	defaultSMTPPort   = "587"
	defaultSMTPFrom   = "idletime <no-reply@localhost>"
	envDatabaseURL    = "DATABASE_URL"
	envHTTPAddr       = "HTTP_ADDR"
	envAppBaseURL     = "APP_BASE_URL"
	envCookieSecure   = "COOKIE_SECURE"
	envTrustProxy     = "TRUST_PROXY_HEADERS"
	envSMTPHost       = "SMTP_HOST"
	envSMTPPort       = "SMTP_PORT"
	envSMTPUsername   = "SMTP_USERNAME"
	envSMTPPassword   = "SMTP_PASSWORD"
	envSMTPFrom       = "SMTP_FROM"
)

// Config holds all runtime settings of the server.
type Config struct {
	HTTPAddr    string
	DatabaseURL string

	// AppBaseURL is the public address of the web app. It is used for links in emails
	// and as the only origin allowed to make state-changing requests.
	AppBaseURL string
	// CookieSecure marks the session cookie as HTTPS-only. Enable it in production.
	CookieSecure bool
	// TrustProxyHeaders makes the server believe X-Real-IP. Enable it only behind a
	// reverse proxy that sets the header.
	TrustProxyHeaders bool

	// SMTP configures outgoing email. Without a host, emails are only logged.
	SMTP mail.SMTPConfig
}

// AppOrigin returns the scheme and host of AppBaseURL, e.g. "http://localhost:3000".
func (c Config) AppOrigin() string {
	u, err := url.Parse(c.AppBaseURL)
	if err != nil {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

// Load reads the configuration from environment variables.
func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:    getenv(envHTTPAddr, defaultHTTPAddr),
		DatabaseURL: os.Getenv(envDatabaseURL),
		AppBaseURL:  getenv(envAppBaseURL, defaultAppBaseURL),
		SMTP: mail.SMTPConfig{
			Host:     os.Getenv(envSMTPHost),
			Port:     getenv(envSMTPPort, defaultSMTPPort),
			Username: os.Getenv(envSMTPUsername),
			Password: os.Getenv(envSMTPPassword),
			From:     getenv(envSMTPFrom, defaultSMTPFrom),
		},
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("%s is required", envDatabaseURL)
	}

	base, err := url.Parse(cfg.AppBaseURL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return Config{}, fmt.Errorf("%s must be an absolute URL, got %q", envAppBaseURL, cfg.AppBaseURL)
	}

	if cfg.CookieSecure, err = getenvBool(envCookieSecure); err != nil {
		return Config{}, err
	}
	if cfg.TrustProxyHeaders, err = getenvBool(envTrustProxy); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getenvBool(key string) (bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return false, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, errors.Join(fmt.Errorf("%s must be a boolean", key), err)
	}
	return parsed, nil
}
