// Package server wires HTTP routes and middleware together.
package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/schellhas/idletime/backend/internal/httpx"
)

const readinessTimeout = 2 * time.Second

// Pinger reports whether a dependency (the database) is reachable.
type Pinger interface {
	Ping(ctx context.Context) error
}

// RouteRegistrar is implemented by every feature handler.
type RouteRegistrar interface {
	Register(mux *http.ServeMux)
}

// Dependencies are the collaborators needed to build the router.
type Dependencies struct {
	Logger     *slog.Logger
	Pinger     Pinger
	Categories RouteRegistrar
}

// NewRouter builds the root handler including middleware.
func NewRouter(deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthHandler)
	mux.HandleFunc("GET /readyz", readyHandler(deps.Pinger))
	deps.Categories.Register(mux)

	return recoverer(deps.Logger)(requestLogger(deps.Logger)(mux))
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func readyHandler(pinger Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
		defer cancel()
		if err := pinger.Ping(ctx); err != nil {
			httpx.WriteError(w, http.StatusServiceUnavailable, "not_ready", "database unavailable")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}
