package server

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubPinger struct{ err error }

func (s stubPinger) Ping(context.Context) error { return s.err }

type noRoutes struct{}

func (noRoutes) Register(*http.ServeMux) {}

func newTestRouter(pingErr error) http.Handler {
	return NewRouter(Dependencies{
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		Pinger:     stubPinger{err: pingErr},
		Categories: noRoutes{},
	})
}

func TestHealthEndpoints(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		pingErr    error
		wantStatus int
	}{
		{name: "liveness", path: "/healthz", wantStatus: http.StatusOK},
		{name: "ready", path: "/readyz", wantStatus: http.StatusOK},
		{name: "not ready", path: "/readyz", pingErr: errors.New("down"), wantStatus: http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			newTestRouter(tt.pingErr).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
