package config

import "testing"

func setBase(t *testing.T) {
	t.Helper()
	t.Setenv(envDatabaseURL, "postgres://example")
	for _, key := range []string{envHTTPAddr, envAppBaseURL, envCookieSecure, envTrustProxy, envSMTPHost, envSMTPPort} {
		t.Setenv(key, "")
	}
}

func TestLoad(t *testing.T) {
	t.Run("requires database url", func(t *testing.T) {
		setBase(t)
		t.Setenv(envDatabaseURL, "")
		if _, err := Load(); err == nil {
			t.Fatal("expected an error when DATABASE_URL is missing")
		}
	})

	t.Run("applies defaults", func(t *testing.T) {
		setBase(t)
		cfg, err := Load()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.HTTPAddr != defaultHTTPAddr || cfg.AppBaseURL != defaultAppBaseURL || cfg.SMTP.Port != defaultSMTPPort {
			t.Errorf("unexpected defaults: %+v", cfg)
		}
		if cfg.CookieSecure || cfg.TrustProxyHeaders {
			t.Error("security-relevant switches must default to off")
		}
	})

	t.Run("honours overrides", func(t *testing.T) {
		setBase(t)
		t.Setenv(envHTTPAddr, ":9999")
		t.Setenv(envCookieSecure, "true")
		t.Setenv(envAppBaseURL, "https://idletime.example.com")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.HTTPAddr != ":9999" || !cfg.CookieSecure || cfg.AppOrigin() != "https://idletime.example.com" {
			t.Errorf("unexpected config: %+v", cfg)
		}
	})

	t.Run("rejects invalid values", func(t *testing.T) {
		setBase(t)
		t.Setenv(envAppBaseURL, "not a url")
		if _, err := Load(); err == nil {
			t.Error("expected an error for an invalid APP_BASE_URL")
		}

		setBase(t)
		t.Setenv(envCookieSecure, "maybe")
		if _, err := Load(); err == nil {
			t.Error("expected an error for an invalid COOKIE_SECURE")
		}
	})
}
