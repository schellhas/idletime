package config

import "testing"

func TestLoad(t *testing.T) {
	t.Run("requires database url", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "")
		if _, err := Load(); err == nil {
			t.Fatal("expected an error when DATABASE_URL is missing")
		}
	})

	t.Run("applies defaults", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://example")
		t.Setenv("HTTP_ADDR", "")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.HTTPAddr != defaultHTTPAddr {
			t.Errorf("HTTPAddr = %q, want %q", cfg.HTTPAddr, defaultHTTPAddr)
		}
	})

	t.Run("honours overrides", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://example")
		t.Setenv("HTTP_ADDR", ":9999")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.HTTPAddr != ":9999" {
			t.Errorf("HTTPAddr = %q, want :9999", cfg.HTTPAddr)
		}
	})
}
