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

const testOrigin = "http://localhost:3000"

type stubPinger struct{ err error }

func (s stubPinger) Ping(context.Context) error { return s.err }

type echoRoutes struct{}

func (echoRoutes) Register(mux *http.ServeMux) {
	ok := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }
	mux.HandleFunc("GET /echo", ok)
	mux.HandleFunc("POST /echo", ok)
}

func newTestRouter(pingErr error) http.Handler {
	return NewRouter(Dependencies{
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		Pinger:        stubPinger{err: pingErr},
		Routes:        []RouteRegistrar{echoRoutes{}},
		AllowedOrigin: testOrigin,
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

func TestOriginCheck(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		origin     string
		wantStatus int
	}{
		{name: "same origin POST", method: http.MethodPost, origin: testOrigin, wantStatus: http.StatusNoContent},
		{name: "no origin header POST", method: http.MethodPost, wantStatus: http.StatusNoContent},
		{name: "foreign origin POST", method: http.MethodPost, origin: "https://evil.example", wantStatus: http.StatusForbidden},
		{name: "foreign origin GET", method: http.MethodGet, origin: "https://evil.example", wantStatus: http.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/echo", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()

			newTestRouter(nil).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
